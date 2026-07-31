package embeddedpg_test

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/embeddedpg"
)

// occupyPort holds a port for the duration of the test, standing in for a
// postmaster that outlived the process which started it.
func occupyPort(t *testing.T) uint32 {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() }) //nolint:errcheck
	return uint32(l.Addr().(*net.TCPAddr).Port)
}

// writePostmasterFile writes a postmaster.pid with the layout Postgres uses:
// pid, data directory, start time, port, then socket details.
func writePostmasterFile(t *testing.T, dir string, pid int, dataDir string, port uint32) {
	t.Helper()
	body := fmt.Sprintf("%d\n%s\n1753800000\n%d\n/tmp\nlocalhost\n5432001         1\nready\n",
		pid, dataDir, port)
	if err := os.WriteFile(filepath.Join(dir, "postmaster.pid"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// liveProcess returns the pid of a running process standing in for a
// postmaster. It must be a direct child so the goroutine below can reap it:
// an unreaped zombie still answers kill(pid, 0), which would hide whether
// Stop worked. Spawning it via `sh -c ... &` would also make it inherit
// SIG_IGN for SIGINT, so only the force-kill backstop could end it.
func liveProcess(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go cmd.Wait()                            //nolint:errcheck
	t.Cleanup(func() { cmd.Process.Kill() }) //nolint:errcheck
	return cmd.Process.Pid
}

func newDataDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "postgres")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Air SIGKILLs the app on every reload, so Postgres routinely outlives the
// process that started it. The next build must attach to it, not refuse.
func TestStartAdoptsItsOwnPostmasterAcrossReload(t *testing.T) {
	port := occupyPort(t)
	dataDir := newDataDir(t)
	writePostmasterFile(t, dataDir, os.Getpid(), dataDir, port)

	srv, err := embeddedpg.Start(embeddedpg.Options{DataDir: dataDir, Port: port})

	if err != nil {
		t.Fatalf("own postmaster must be adopted, got: %v", err)
	}
	if !strings.Contains(srv.DSN, strconv.Itoa(int(port))) {
		t.Errorf("DSN should point at the adopted port, got %s", srv.DSN)
	}
}

// The bug this guards: two projects both defaulting to one port meant the
// second adopted the first's cluster and ran its migrations into it.
func TestStartRefusesAPostmasterForAnotherDataDir(t *testing.T) {
	port := occupyPort(t)
	dataDir := newDataDir(t)
	writePostmasterFile(t, dataDir, os.Getpid(), "/somewhere/else/postgres", port)

	srv, err := embeddedpg.Start(embeddedpg.Options{DataDir: dataDir, Port: port})

	if srv != nil {
		t.Fatalf("another checkout's database must never be adopted, got %+v", srv)
	}
	if !errors.Is(err, embeddedpg.ErrPortInUse) {
		t.Fatalf("want ErrPortInUse, got %v", err)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(int(port))) {
		t.Errorf("error should name the port, got: %v", err)
	}
}

func TestStartRefusesAnOccupantWithNoPostmasterFile(t *testing.T) {
	port := occupyPort(t)
	dataDir := filepath.Join(t.TempDir(), "postgres")

	srv, err := embeddedpg.Start(embeddedpg.Options{DataDir: dataDir, Port: port})

	if srv != nil {
		t.Fatalf("an unidentified occupant must not be adopted, got %+v", srv)
	}
	if !errors.Is(err, embeddedpg.ErrPortInUse) {
		t.Fatalf("want ErrPortInUse, got %v", err)
	}
	if _, statErr := os.Stat(dataDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("a refused start must not create %s", dataDir)
	}
}

func TestStartRefusesADeadPostmaster(t *testing.T) {
	port := occupyPort(t)
	dataDir := newDataDir(t)
	writePostmasterFile(t, dataDir, 0x7FFFFFFE, dataDir, port)

	_, err := embeddedpg.Start(embeddedpg.Options{DataDir: dataDir, Port: port})

	if !errors.Is(err, embeddedpg.ErrPortInUse) {
		t.Fatalf("a stale pid file must not authorise adoption, got %v", err)
	}
}

// Our own instance recorded on a different port means the thing holding this
// port is somebody else, however familiar the data dir looks.
func TestStartRefusesWhenOurPostmasterRunsElsewhere(t *testing.T) {
	port := occupyPort(t)
	dataDir := newDataDir(t)
	writePostmasterFile(t, dataDir, os.Getpid(), dataDir, port+1)

	_, err := embeddedpg.Start(embeddedpg.Options{DataDir: dataDir, Port: port})

	if !errors.Is(err, embeddedpg.ErrPortInUse) {
		t.Fatalf("want ErrPortInUse, got %v", err)
	}
}

// A failed startup calls Stop on its way out. An adopted postmaster belongs
// to whoever launched it — very likely a dev server still using it — so Stop
// must leave it alone rather than take the database down with it.
func TestStopLeavesAnAdoptedPostmasterRunning(t *testing.T) {
	port := occupyPort(t)
	dataDir := newDataDir(t)
	pid := liveProcess(t)
	writePostmasterFile(t, dataDir, pid, dataDir, port)

	srv, err := embeddedpg.Start(embeddedpg.Options{DataDir: dataDir, Port: port})
	if err != nil {
		t.Fatalf("adoption failed: %v", err)
	}
	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	if err := syscall.Kill(pid, 0); err != nil {
		t.Errorf("Stop killed adopted postmaster pid %d: %v", pid, err)
	}
}
