package server

import (
	"net/http"
	"strings"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/assets"
	"{{ cookiecutter.go_module_path.strip() }}/internal/logging"
	"{{ cookiecutter.go_module_path.strip() }}/internal/version"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	_ "github.com/danielgtaylor/huma/v2/formats/cbor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *App) routes() http.Handler {
	router := chi.NewMux()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(logging.RequestLogger(logging.HTTPLoggerConfig{
		Logger:      app.Log,
		Concise:     app.Conf.AppConf.LogConcise,
		QuietRoutes: []logging.QuietRoute{% raw %}{{Pattern: "/healthz", Period: 10 * time.Second}}{% endraw %},
		SkipPaths:   []string{"/static/"},
	}))

	humaCfg := huma.DefaultConfig("{{ cookiecutter.project_name }}", version.Get())
	humaCfg.DocsPath = ""
	humaCfg.Info.Description = `## Overview

{{ cookiecutter.project_description }}

## Authentication

| Method | Header | Notes |
|---|---|---|
| API Key | ` + "`X-API-Key`" + ` header | Set via X_API_KEY env var |`
	humaCfg.Tags = []*huma.Tag{
		{Name: "Monitoring", Description: "Service health and version."},
	}
	humaCfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"xApiKey": {Type: "apiKey", In: "header", Name: "X-API-Key"},
	}
	api := humachi.New(router, humaCfg)

	router.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		csp := []string{
			"default-src 'none'",
			"connect-src 'self'",
			"sandbox allow-same-origin allow-scripts",
			"script-src 'unsafe-inline' https://unpkg.com/@scalar/api-reference@1.48.0/dist/browser/standalone.js",
			"style-src 'unsafe-inline'",
			"worker-src blob:",
		}
		w.Header().Set("Content-Security-Policy", strings.Join(csp, "; "))
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <title>API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script id="api-reference" data-url="/openapi.json"></script>
    <script src="https://unpkg.com/@scalar/api-reference@1.48.0/dist/browser/standalone.js" crossorigin></script>
  </body>
</html>`))
	})

	app.registerEndpoints(api)
	fileServer := http.FileServer(http.FS(assets.EmbeddedFiles))
	router.Handle("/static/*", fileServer)

	return router
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
