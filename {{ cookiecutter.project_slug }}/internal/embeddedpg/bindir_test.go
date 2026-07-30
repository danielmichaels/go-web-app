package embeddedpg

import (
	"strings"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// The binaries directory must be version-namespaced.
//
// embedded-postgres skips extraction whenever <binariesPath>/bin/pg_ctl
// already exists and never checks which version is there. Share one directory
// across versions and a project asking for V16 silently runs whichever major
// version another project extracted first. initdb then stamps the real
// version into PG_VERSION, the library compares it against the requested
// version on the next start, decides the data directory is foreign, and
// deletes it — a dev database wiped on every restart, with no error logged.
func TestStableBinDirIsVersionNamespaced(t *testing.T) {
	v16, err := stableBinDir(embeddedpostgres.V16)
	if err != nil {
		t.Fatalf("stableBinDir(V16): %v", err)
	}
	v17, err := stableBinDir(embeddedpostgres.V17)
	if err != nil {
		t.Fatalf("stableBinDir(V17): %v", err)
	}

	if v16 == v17 {
		t.Fatalf("V16 and V17 share the binaries dir %q; one will run the other's binaries", v16)
	}
	if !strings.Contains(v16, string(embeddedpostgres.V16)) {
		t.Errorf("path %q does not contain the version %q", v16, embeddedpostgres.V16)
	}
}
