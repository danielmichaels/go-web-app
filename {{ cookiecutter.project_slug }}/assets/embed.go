// Package assets holds everything compiled into the binary: the goose
// migrations applied at boot{% if not cookiecutter.api_only %}, and the static files served under
// /static{% endif %}. Nothing here needs to exist on the host at runtime.
package assets

import (
	"embed"
)

{% if cookiecutter.api_only -%}
//go:embed migrations
{% else -%}
//go:embed migrations static
{% endif -%}
var EmbeddedFiles embed.FS
