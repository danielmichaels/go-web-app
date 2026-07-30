package server

import (
	"log/slog"
	"net/http"
{% if cookiecutter.use_river and not cookiecutter.api_only -%}
	"strings"
{% endif -%}

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v3"
)

// ApiKeyAuth checks the X-API-Key header. The key is read from the App's
// config rather than the package-level loader so a test can supply its own.
func (app *App) ApiKeyAuth(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if ctx.Header("X-API-Key") != app.Conf.Server.XApiKey {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(ctx)
	}
}

// httplogOptions configures the access log: a one-line summary for local dev,
// the full attribute set for JSON.
//
// Panics are httplog's to recover so they are logged with the request
// attached — see the middleware order in Routes.
//
// LogExtraAttrs is deliberately unset: a non-nil value makes httplog tee every
// request body into an unbounded buffer, which would hold whole uploads in
// memory.
func (app *App) httplogOptions() *httplog.Options {
	o := &httplog.Options{
		Level:         slog.LevelInfo,
		Schema:        httplog.SchemaOTEL.Concise(!app.Conf.AppConf.LogJson),
		RecoverPanics: true,
		Skip:          app.skipRequestLog,
	}
	// Concise blanks the values but keeps the header keys, so headers are only
	// worth collecting when the full schema is in play.
	if app.Conf.AppConf.LogJson {
		if app.Conf.AppConf.LogRequestHeaders {
			o.LogRequestHeaders = []string{"Origin"}
		}
		if app.Conf.AppConf.LogResponseHeaders {
			o.LogResponseHeaders = []string{"Content-Type"}
		}
	}
	return o
}

// skipRequestLog keeps static assets and health polling out of the access log.
// httplog calls this after the handler returns, so the chi route pattern has
// resolved by now.
func (app *App) skipRequestLog(r *http.Request, _ int) bool {
	route := chi.RouteContext(r.Context()).RoutePattern()
	switch route {
	case "/static/*", "/healthz":
		return true
	}
{% if cookiecutter.use_river and not cookiecutter.api_only -%}
	// The job dashboard polls its own API constantly.
	if route != "" && strings.HasPrefix(route, app.Conf.AppConf.RiverUIPath) {
		return true
	}
{% endif -%}
	return false
}
