package server

import (
	"github.com/danielgtaylor/huma/v2"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"net/http"
)

func ApiKeyAuth(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if ctx.Header("X-API-Key") != config.AppConfig().Server.XApiKey {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(ctx)
	}
}
