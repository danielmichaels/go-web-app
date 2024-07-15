package webserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"{{ cookiecutter.go_module_path.strip() }}/assets"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
)

func (app *Application) routes() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.RealIP)
	router.Use(middleware.Compress(5))
	router.Use(httplog.RequestLogger(*app.Logger, []string{
		"/healthz",
		"/static/img/logo.png",
		"/static/css/theme.css",
		"/static/js/bundle.js",
		"/static/js/htmx.min.js",
	}))
	router.Use(middleware.Heartbeat("/healthz"))

	router.NotFound(app.notFound)
	router.MethodNotAllowed(app.methodNotAllowed)

	fileServer := http.FileServer(http.FS(assets.EmbeddedAssets))
	router.Handle("/static/*", fileServer)

	return router
}
