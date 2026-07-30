package embeddedpg

import (
	"os"
	"path/filepath"
	"time"
)

// sweepGrace is how long a directory must have been untouched before the
// sweep will consider it abandoned. `go test ./...` runs packages in
// parallel, so a directory created seconds ago may belong to an instance
// that has not written its postmaster.pid yet.
const sweepGrace = 10 * time.Minute

// SweepResult counts what a sweep did. Kept is the directories that look
// like they are still in use.
type SweepResult struct {
	Removed int
	Kept    int
}

// Sweep removes embedded-postgres temp directories left behind by runs that
// were killed before Stop could run — a SIGKILL, a `go test -timeout`, or a
// panicking test binary. It is safe to call while other instances are
// running: a data directory whose postmaster is still alive is kept, as is
// a runtime directory that still holds a socket.
//
// It does not kill anything. An orphaned postmaster still holds its data
// directory, so freeing that disk needs `task clean:pg FORCE=1`.
func Sweep() SweepResult { return SweepDir(os.TempDir()) }

// SweepDir is Sweep against an explicit parent directory.
func SweepDir(parent string) SweepResult {
	var res SweepResult
	for _, pattern := range []string{"embeddedpg-data-*", "embeddedpg-rt-*"} {
		matches, err := filepath.Glob(filepath.Join(parent, pattern))
		if err != nil {
			continue
		}
		for _, dir := range matches {
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				continue
			}
			if time.Since(info.ModTime()) < sweepGrace {
				continue
			}
			if inUse(dir) {
				res.Kept++
				continue
			}
			if os.RemoveAll(dir) == nil {
				res.Removed++
			}
		}
	}
	return res
}

// inUse reports whether dir looks like it belongs to a running instance: a
// data directory with a live postmaster, or a runtime directory that still
// has a socket in it.
func inUse(dir string) bool {
	if pid, err := readPostmasterPID(dir); err == nil {
		return processAlive(pid)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true // unreadable: leave it alone
	}
	for _, e := range entries {
		if len(e.Name()) >= 9 && e.Name()[:9] == ".s.PGSQL." {
			return true
		}
	}
	return false
}
