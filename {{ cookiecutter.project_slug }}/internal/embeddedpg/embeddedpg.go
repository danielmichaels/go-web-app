// Package embeddedpg runs Postgres as a child of this process: a throwaway
// instance for tests, or a persistent one for local dev. Either way
// `go test ./...` and `task dev` need no installed or running Postgres.
package embeddedpg

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// Options controls the embedded Postgres instance. The zero value is
// suitable for ephemeral test use: temp data dir, random free port, and
// postgres/postgres credentials.
type Options struct {
	// Logger routes embedded-postgres process output (initdb, pg_ctl)
	// through slog at Debug level. Nil = discard.
	Logger   *slog.Logger
	User     string
	Password string
	Database string
	// DataDir is the directory for Postgres data files. Empty = temp dir
	// (removed on Stop). Set it for persistent dev use.
	DataDir string
	Version embeddedpostgres.PostgresVersion
	Port    uint32
}

// Server is a running embedded Postgres instance.
type Server struct {
	DSN     string
	pg      *embeddedpostgres.EmbeddedPostgres
	dataDir string
	// owned temp dirs cleaned up on Stop
	ownedDataDir    string
	ownedRuntimeDir string
	// adopted marks a postmaster that outlived the process which launched
	// it, so this Server has no library handle to stop it through.
	adopted bool
}

func (o *Options) applyDefaults() error {
	if o.User == "" {
		o.User = "postgres"
	}
	if o.Password == "" {
		o.Password = "postgres"
	}
	if o.Database == "" {
		o.Database = "test-db"
	}
	if o.Version == "" {
		o.Version = embeddedpostgres.V16
	}
	if o.Port == 0 {
		p, err := freePort()
		if err != nil {
			return fmt.Errorf("embeddedpg: find free port: %w", err)
		}
		o.Port = uint32(p)
	}
	return nil
}

// stableBinDir returns the shared, stable extraction dir for a version.
// Setting BinariesPath here prevents concurrent Start() calls from deleting
// each other's extracted files (the default RuntimePath is also this dir and
// is rm-rf'd on every Start).
//
// The version is part of the path, and must stay that way. embedded-postgres
// skips extraction whenever <binariesPath>/bin/pg_ctl already exists, without
// checking which version is there — so a single shared directory hands every
// project whichever major version happened to extract first. The mismatch is
// silent until restart: initdb stamps the real version into PG_VERSION, the
// library compares that against the *requested* version, decides the data
// directory is foreign, and deletes it. That is a wiped dev database on every
// start, with no error anywhere.
func stableBinDir(version embeddedpostgres.PostgresVersion) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("embeddedpg: home dir: %w", err)
	}
	return filepath.Join(home, ".embedded-postgres-go", string(version), "extracted"), nil
}

// isOwnPostmaster reports whether the process holding port is the postmaster
// for dataDir. postmaster.pid records both facts — the data directory on line
// 2 and the port on line 4 — so no connection is needed to tell this
// instance's own leftover from a different checkout sharing the port.
func isOwnPostmaster(dataDir string, port uint32) bool {
	if dataDir == "" {
		return false
	}
	lines, err := readPostmasterFile(dataDir)
	if err != nil || len(lines) < 4 {
		return false
	}
	pid, err := strconv.Atoi(lines[0])
	if err != nil || !processAlive(pid) {
		return false
	}
	recordedPort, err := strconv.Atoi(lines[3])
	if err != nil || uint32(recordedPort) != port {
		return false
	}
	return sameDir(lines[1], dataDir)
}

func readPostmasterFile(dataDir string) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(dataDir, "postmaster.pid"))
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
}

// sameDir compares paths through symlinks, because macOS resolves the temp
// dirs used for data directories via /private.
func sameDir(a, b string) bool {
	resolve := func(p string) string {
		abs, err := filepath.Abs(p)
		if err != nil {
			return filepath.Clean(p)
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			return real
		}
		return abs
	}
	return resolve(a) == resolve(b)
}

func portConflict(port uint32, dataDir string) error {
	if dataDir == "" {
		return fmt.Errorf("embeddedpg: %w: %d", ErrPortInUse, port)
	}
	return fmt.Errorf(
		"embeddedpg: %w: %d is held by a postmaster that is not the one for %s "+
			"(another checkout, or a stale instance); stop it or set EMBEDDED_POSTGRES_PORT",
		ErrPortInUse, port, dataDir,
	)
}

// ErrPortInUse reports that the port is held by a postmaster this instance
// cannot prove is its own. Adopting one blindly makes two checkouts share a
// database, and the difference only surfaces later as errors against tables
// that look correct on disk.
var ErrPortInUse = errors.New("port already in use")

// portInUse reports whether something already accepts connections on port.
func portInUse(port uint32) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return true
	}
	l.Close()
	return false
}

// Start launches an embedded Postgres instance and returns once it is
// ready. With an empty DataDir a temp directory is created and removed on
// Stop. Each call uses its own RuntimePath so parallel instances do not
// race on extraction.
//
// A persistent instance (DataDir set) whose port is already held by its own
// postmaster is adopted rather than restarted — air SIGKILLs the app on
// reload, so Postgres routinely outlives it. Any other occupant is fatal;
// see ErrPortInUse.
func Start(opts Options) (*Server, error) {
	if err := opts.applyDefaults(); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@localhost:%d/%s?sslmode=disable",
		opts.User, opts.Password, opts.Port, opts.Database,
	)

	if portInUse(opts.Port) {
		if !isOwnPostmaster(opts.DataDir, opts.Port) {
			return nil, portConflict(opts.Port, opts.DataDir)
		}
		return &Server{DSN: dsn, dataDir: opts.DataDir, adopted: true}, nil
	}

	// Reclaim what earlier runs could not: Stop never runs under SIGKILL,
	// `go test -timeout`, or a panicking test binary.
	Sweep()

	var ownedDataDir string
	dataDir := opts.DataDir
	if dataDir == "" {
		tmp, err := os.MkdirTemp("", "embeddedpg-data-*")
		if err != nil {
			return nil, fmt.Errorf("embeddedpg: mkdirtemp data: %w", err)
		}
		dataDir = tmp
		ownedDataDir = tmp
	}

	// A persistent instance gets a runtime dir beside its data, reused every
	// run: a temp one would be abandoned on every dev restart.
	var runtimeDir, ownedRuntimeDir string
	if opts.DataDir != "" {
		runtimeDir = filepath.Join(filepath.Dir(opts.DataDir), "runtime")
		if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
			return nil, fmt.Errorf("embeddedpg: runtime dir: %w", err)
		}
	} else {
		tmp, err := os.MkdirTemp("", "embeddedpg-rt-*")
		if err != nil {
			os.RemoveAll(ownedDataDir) //nolint:errcheck
			return nil, fmt.Errorf("embeddedpg: mkdirtemp runtime: %w", err)
		}
		runtimeDir, ownedRuntimeDir = tmp, tmp
	}

	binDir, err := stableBinDir(opts.Version)
	if err != nil {
		if ownedDataDir != "" {
			os.RemoveAll(ownedDataDir) //nolint:errcheck
		}
		os.RemoveAll(ownedRuntimeDir) //nolint:errcheck
		return nil, err
	}

	cfg := embeddedpostgres.DefaultConfig().
		Username(opts.User).
		Password(opts.Password).
		Database(opts.Database).
		Port(opts.Port).
		DataPath(dataDir).
		RuntimePath(runtimeDir).
		BinariesPath(binDir).
		Version(opts.Version).
		Logger(&pgSlogWriter{l: opts.Logger})

	pg := embeddedpostgres.NewDatabase(cfg)
	if err := pg.Start(); err != nil {
		if ownedDataDir != "" {
			os.RemoveAll(ownedDataDir) //nolint:errcheck
		}
		os.RemoveAll(ownedRuntimeDir) //nolint:errcheck
		return nil, fmt.Errorf("embeddedpg: start: %w", err)
	}

	return &Server{
		DSN:             dsn,
		pg:              pg,
		dataDir:         dataDir,
		ownedDataDir:    ownedDataDir,
		ownedRuntimeDir: ownedRuntimeDir,
	}, nil
}

// Stop shuts down the instance and removes its temp directories.
//
// pg_ctl stop -w is not fully reliable: it can report success — and
// embeddedpostgres.Stop() returns no error — while the postmaster process
// is still alive. So Stop reads the postmaster PID before asking the
// library to stop, and force-kills it afterward if it is still running.
func (s *Server) Stop() error {
	// An adopted postmaster was launched by someone else and may still be
	// serving them; a caller that never started it does not get to end it.
	// It survives as an orphan, which the next run in this data dir adopts.
	if s.adopted {
		return nil
	}
	if s.pg != nil {
		pid, pidErr := readPostmasterPID(s.dataDir)

		stopErr := s.pg.Stop()

		if pidErr == nil {
			if killErr := ensureProcessTerminated(pid, 5*time.Second); killErr != nil {
				return fmt.Errorf("embeddedpg: stop: %w", killErr)
			}
		}

		if stopErr != nil && pidErr != nil {
			return fmt.Errorf("embeddedpg: stop: %w", stopErr)
		}
	}
	if s.ownedDataDir != "" {
		os.RemoveAll(s.ownedDataDir) //nolint:errcheck
	}
	if s.ownedRuntimeDir != "" {
		os.RemoveAll(s.ownedRuntimeDir) //nolint:errcheck
	}
	return nil
}

// readPostmasterPID reads the postmaster PID from <dataDir>/postmaster.pid.
// It must be read before calling embeddedpostgres.Stop() — Postgres removes
// this file early in its shutdown sequence.
func readPostmasterPID(dataDir string) (int, error) {
	b, err := os.ReadFile(filepath.Join(dataDir, "postmaster.pid"))
	if err != nil {
		return 0, err
	}
	line, _, _ := bytes.Cut(b, []byte("\n"))
	pid, err := strconv.Atoi(string(line))
	if err != nil {
		return 0, fmt.Errorf("parse postmaster.pid: %w", err)
	}
	return pid, nil
}

// ensureProcessTerminated polls pid for up to timeout, then SIGKILLs it if
// it is still alive.
func ensureProcessTerminated(pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !processAlive(pid) {
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("pid %d survived pg_ctl stop and SIGKILL failed: %w", pid, err)
	}
	time.Sleep(200 * time.Millisecond)
	if processAlive(pid) {
		return fmt.Errorf("pid %d still alive after SIGKILL", pid)
	}
	return nil
}

func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// pgSlogWriter routes embedded-postgres process output to slog at Debug
// level. When l is nil all output is discarded.
type pgSlogWriter struct{ l *slog.Logger }

func (w *pgSlogWriter) Write(p []byte) (int, error) {
	if w.l != nil {
		for line := range strings.SplitSeq(strings.TrimRight(string(p), "\n"), "\n") {
			if line != "" {
				w.l.Debug(line)
			}
		}
	}
	return len(p), nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}
