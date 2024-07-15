package assets

import "embed"

//go:embed "static" "migrations"
var EmbeddedAssets embed.FS
