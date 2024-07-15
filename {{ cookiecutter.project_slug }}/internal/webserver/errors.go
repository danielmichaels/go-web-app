package webserver

import (
	"net/http"

	"{{ cookiecutter.go_module_path.strip() }}/assets/static/view/pages"
	"{{ cookiecutter.go_module_path.strip() }}/internal/render"
)
func (app *Application) notFound(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	switch accept {
	default:
		_ = render.Render(r.Context(), w, http.StatusNotFound, pages.NotFoundErrorPage())
	}
}

func (app *Application) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	switch accept {
	default:
		_ = render.Render(
			r.Context(),
			w,
			http.StatusInternalServerError,
			pages.InternalServerErrorPage(),
		)
	}
}

//func (app *Application) serverError(w http.ResponseWriter, r *http.Request, err error) {
//	app.Logger.Error().Err(err).Send()
//	chirender.HTML(w, r, "<h2>ERROR</h2>")
//}
