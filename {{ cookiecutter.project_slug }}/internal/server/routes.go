package server

import (
	"{{ cookiecutter.go_module_path.strip() }}/internal/version"
	"net/http"

	"github.com/danielgtaylor/huma/v2/autopatch"

	"{{ cookiecutter.go_module_path.strip() }}/assets"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	_ "github.com/danielgtaylor/huma/v2/formats/cbor"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
)

func (app *Server) routes() http.Handler {
	router := chi.NewMux()
	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.Compress(5))
	router.Use(httplog.RequestLogger(httpLogger(app.Conf)))

	cfg := huma.DefaultConfig("{{ '-'.join(cookiecutter['project_name'].lower().split()).replace('_', '-') }}", version.Get())
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"xApiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
	}
	api := humachi.New(router, cfg)
	autopatch.AutoPatch(api)

	cfg.Info.Title = "gecko API"
	cfg.Info.Description = "API for the gecko application"

	router.Get("/scalar", app.handleScalarDocsGet)

	app.registerEndpoints(api)
	fileServer := http.FileServer(http.FS(assets.EmbeddedFiles))
	router.Handle("/static/*", fileServer)

	return router
}

// handleScalarDocsGet is an HTTP handler that serves the API reference documentation
// for the application. It writes an HTML page that includes a script tag that loads
// the Scalar API reference viewer, which will fetch the OpenAPI specification from
// the "/openapi.json" endpoint.
func (app *Server) handleScalarDocsGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(`<!doctype html>
<html>
  <head>
    <title>API Reference</title>
    <meta charset="utf-8" />
    <meta
      name="viewport"
      content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/openapi.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`))
}

func (app *Server) registerEndpoints(api huma.API) {
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
