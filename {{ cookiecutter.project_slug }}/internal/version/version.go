package version

import (
	"runtime/debug"
)

// Version and Revision are injected at build time with
// -ldflags "-X <module>/internal/version.Version=...". Both stay empty for a
// plain `go build`, which falls back to the VCS information the toolchain
// embeds automatically.
var (
	Version  string
	Revision string
)

const (
	unavailable = "unavailable"
	// revisionLength trims a full git SHA to something readable in logs.
	revisionLength = 12
)

// Get returns the build version: the CI-injected tag if there is one, else the
// commit it was built from, suffixed -dirty when the tree had local changes.
func Get() string {
	if Version != "" {
		return Version
	}

	revision, modified := Revision, false
	if revision == "" {
		revision, modified = fromBuildInfo()
	}
	if revision == "" {
		return unavailable
	}
	if len(revision) > revisionLength {
		revision = revision[:revisionLength]
	}
	if modified {
		return revision + "-dirty"
	}
	return revision
}

func fromBuildInfo() (revision string, modified bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return revision, modified
}
