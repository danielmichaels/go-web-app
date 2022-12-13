package assets

import (
	"embed"
)

// go:embed "emails" "migrations" "templates" "static"
//
//go:embed "emails" "templates" "static"
var EmbeddedFiles embed.FS
