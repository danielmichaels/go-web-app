package assets

import (
	"embed"
)

//go:embed "emails" "migrations" "templates" "static"
var EmbeddedFiles embed.FS
