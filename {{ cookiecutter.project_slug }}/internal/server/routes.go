package server

import (
{% if not cookiecutter.api_only -%}
	"io/fs"
{% endif -%}
	"net/http"

{% if not cookiecutter.api_only -%}
	"{{ cookiecutter.go_module_path.strip() }}/assets"
	"{{ cookiecutter.go_module_path.strip() }}/internal/ui"
{% endif -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/version"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	_ "github.com/danielgtaylor/huma/v2/formats/cbor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

// Routes builds the full router. Kept as one method so httptest suites
// exercise the exact production middleware stack.
{% if not cookiecutter.api_only -%}
//
// Static assets are registered before the session middleware: LoadAndSave
// touches the session store on every request it wraps, and a page full of
// assets would otherwise turn one navigation into a burst of session writes.
{% endif -%}
func (app *App) Routes() http.Handler {
	router := chi.NewMux()
	router.Use(middleware.RequestID)
	// The template cannot know which reverse proxies a generated deployment
	// trusts. Never infer a client IP from forwarded headers by default: those
	// headers are client-controlled when the app is directly reachable. A
	// deployment behind a proxy must replace this with a specifically trusted
	// ClientIPFromHeader or ClientIPFromXFF configuration.
	router.Use(middleware.ClientIPFromRemoteAddr)
	// Recoverer sits outside the access log purely as a backstop for httplog
	// itself: httplog recovers handler panics first and logs them with the
	// request and a stack trace, which Recoverer alone cannot do.
	router.Use(middleware.Recoverer)
	router.Use(httplog.RequestLogger(app.Log, app.httplogOptions()))
	router.Use(middleware.Compress(5))
	router.Use(securityHeaders)

{% if not cookiecutter.api_only -%}
	staticFS, err := fs.Sub(assets.EmbeddedFiles, "static")
	if err != nil {
		panic(err)
	}
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

{% if cookiecutter.use_pwa -%}
	// Served from the root, not /static: a service worker only controls the
	// path it was served from. Outside the session group for the same reason
	// the static tree is — it is public and fetched on every visit.
	router.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		// The worker is how a new deployment reaches an installed app, so it
		// is the one file that must never come from a browser cache.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, staticFS, "sw.js")
	})
	router.Get("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		http.ServeFileFS(w, r, staticFS, "manifest.json")
	})
{% endif -%}
{% endif -%}

	api := humachi.New(router, app.humaConfig())
	app.registerEndpoints(api)

{% if not cookiecutter.api_only -%}
	router.Group(func(r chi.Router) {
		r.Use(app.csrf.Handler)
		r.Use(app.Sessions.LoadAndSave)

		r.Mount("/app", ui.New(ui.Deps{
			Conf:     app.Conf,
			Log:      app.Log,
			Db:       app.Db,
			Sessions: app.Sessions,
		}).Routes())

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/app", http.StatusFound)
		})

{% if cookiecutter.use_river -%}
		// Mounted at the top level, not under /app: the dashboard builds its
		// asset URLs from the prefix it was constructed with, and chi strips a
		// subrouter's mount path before the handler sees it.
		//
		// The nil check keeps Jobs optional: a test that only exercises the
		// HTTP surface should not have to start a job client.
		if app.Jobs != nil && app.Conf.AppConf.RiverUIEnabled {
			r.Group(func(r chi.Router) {
				// This group exists to be gated. The dashboard can cancel and
				// retry jobs, and it inherits sessions and CSRF from the
				// enclosing group but nothing that authorises anyone, so add
				// this app's admin check here:
				//
				//	r.Use(app.requireAdmin)
				//
				// Until then RIVER_UI_EMBEDDED is the only thing standing in
				// front of it, which is why it defaults to false.
				r.Mount(app.Conf.AppConf.RiverUIPath, app.Jobs.UIHandler())
			})
		}
{% endif -%}
	})
{% endif -%}

	return router
}

func (app *App) humaConfig() huma.Config {
	cfg := huma.DefaultConfig("{{ cookiecutter.project_name }}", version.Get())
	// Scalar over the default Stoplight Elements. huma serves it at DocsPath
	// with its own CSP and a subresource-integrity hash on the bundle.
	cfg.DocsRenderer = huma.DocsRendererScalar
	cfg.Info.Description = `## Overview

{{ cookiecutter.project_description }}

## Authentication

| Method | Header | Notes |
|---|---|---|
| API Key | ` + "`X-API-Key`" + ` header | Set via X_API_KEY env var |`
	cfg.Tags = []*huma.Tag{
		{Name: "Monitoring", Description: "Service health and version."},
	}
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"xApiKey": {Type: "apiKey", In: "header", Name: "X-API-Key"},
	}
	return cfg
}

// securityHeaders: no-referrer keeps token-bearing URLs out of the Referer
// header on any outbound navigation.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (app *App) registerEndpoints(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "healthz",
		Method:        http.MethodGet,
		Path:          "/healthz",
		Summary:       "health check",
		Description:   "health check endpoint",
		DefaultStatus: http.StatusOK,
		Tags:          []string{"Monitoring"},
	}, app.handleHealthzGet)

	huma.Register(api, huma.Operation{
		OperationID: "version",
		Method:      http.MethodGet,
		Path:        "/version",
		Summary:     "Server version information",
		Description: "Return the version of the application.",
		Tags:        []string{"Monitoring"},
	}, app.handleVersionGet)
}
