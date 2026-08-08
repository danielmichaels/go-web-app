{% if cookiecutter.use_river %}
package jobs

import (
	"strings"
	"testing"
)

// River elects its leader by client ID and rejects nothing, so a collision
// between two processes is silent and only shows up as jobs running twice.
func TestClientIDDiffersPerCall(t *testing.T) {
	seen := make(map[string]bool, 100)
	for range 100 {
		id := clientID()
		if seen[id] {
			t.Fatalf("clientID returned %q twice: two processes could share a leader lease", id)
		}
		seen[id] = true
	}
}

func TestClientIDIsReadable(t *testing.T) {
	id := clientID()

	if len(id) > 40 {
		t.Errorf("clientID = %q (%d chars), want something short enough to sit on a log line", id, len(id))
	}
	if strings.ContainsAny(id, " \t\n=") {
		t.Errorf("clientID = %q, want no whitespace or = so logfmt keeps it one field", id)
	}
	if strings.Contains(id, ".") {
		t.Errorf("clientID = %q, want the host truncated at its first label", id)
	}
}
{% endif %}
