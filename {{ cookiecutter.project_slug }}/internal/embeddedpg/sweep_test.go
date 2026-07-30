package embeddedpg_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/embeddedpg"
)

func mkDir(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// backdate pushes a directory past the sweep's grace period so it looks
// abandoned. It must run after anything is written inside, because creating
// a file updates the parent directory's mtime.
func backdate(t *testing.T, dirs ...string) {
	t.Helper()
	old := time.Now().Add(-2 * time.Hour)
	for _, d := range dirs {
		if err := os.Chtimes(d, old, old); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSweepRemovesDeadLeftoversAndKeepsLiveOnes(t *testing.T) {
	tmp := t.TempDir()

	// A data dir whose run was killed: no postmaster.pid at all.
	noPID := mkDir(t, filepath.Join(tmp, "embeddedpg-data-noPID"))

	// A data dir naming a process that no longer exists.
	deadPID := mkDir(t, filepath.Join(tmp, "embeddedpg-data-deadPID"))
	writePID(t, deadPID, 0x7FFFFFFE)

	// A data dir whose postmaster is this very process: must survive.
	livePID := mkDir(t, filepath.Join(tmp, "embeddedpg-data-livePID"))
	writePID(t, livePID, os.Getpid())

	// An empty runtime dir is a leftover; one holding a socket is in use.
	emptyRT := mkDir(t, filepath.Join(tmp, "embeddedpg-rt-empty"))
	busyRT := mkDir(t, filepath.Join(tmp, "embeddedpg-rt-busy"))
	if err := os.WriteFile(filepath.Join(busyRT, ".s.PGSQL.5433"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	// Not ours: never touched, whatever its age.
	other := mkDir(t, filepath.Join(tmp, "some-other-tool"))

	backdate(t, noPID, deadPID, livePID, emptyRT, busyRT, other)

	// Anything recent belongs to a run that may still be starting up, so it
	// is left alone even though it has no postmaster.pid.
	fresh := mkDir(t, filepath.Join(tmp, "embeddedpg-data-fresh"))

	got := embeddedpg.SweepDir(tmp)

	if got.Removed != 3 {
		t.Fatalf("removed = %d, want 3", got.Removed)
	}
	if got.Kept != 2 {
		t.Fatalf("kept = %d, want 2", got.Kept)
	}
	for _, d := range []string{noPID, deadPID, emptyRT} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed", filepath.Base(d))
		}
	}
	for _, d := range []string{livePID, busyRT, fresh, other} {
		if _, err := os.Stat(d); err != nil {
			t.Errorf("%s should have survived: %v", filepath.Base(d), err)
		}
	}
}

func writePID(t *testing.T, dir string, pid int) {
	t.Helper()
	body := strconv.Itoa(pid) + "\n/some/data/dir\n"
	if err := os.WriteFile(filepath.Join(dir, "postmaster.pid"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
