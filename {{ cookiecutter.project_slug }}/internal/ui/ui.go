// Package ui serves the server-rendered pages: templ for HTML, Datastar for
// interactivity. The JSON API lives in internal/server alongside it and the
// two share nothing but the store.
package ui

import (
	"log/slog"
	"net/http"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
)

// flashKey holds a one-shot message read on the next render.
const flashKey = "flash"

// Deps is everything the UI needs. Sessions is owned by internal/server so a
// single manager wraps both the pages and anything else that needs a session.
type Deps struct {
	Conf     *config.Conf
	Log      *slog.Logger
	Db       *store.Queries
	Sessions *scs.SessionManager
}

type Handlers struct {
	Deps
}

func New(d Deps) *Handlers {
	return &Handlers{Deps: d}
}

// Routes returns the subrouter mounted under /app. The session and CSRF
// middleware are applied by the caller, so everything here already has both.
func (h *Handlers) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.handleHome)
	r.Get("/welcome", h.handleWelcome)
	r.Get("/stream", h.handleStream)
	r.Post("/examples", h.handleExampleCreate)

	return r
}

// serverError logs the cause and shows the visitor nothing about it.
func (h *Handlers) serverError(w http.ResponseWriter, r *http.Request, err error, msg string) {
	h.Log.ErrorContext(r.Context(), msg, "error", err, "path", r.URL.Path)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
