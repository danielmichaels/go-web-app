package server

import (
	"{{ cookiecutter.go_module_path.strip() }}/internal/version"
	"net/http"
)

func (app *Application) newTemplateData(r *http.Request) map[string]any {
	data := map[string]any{
// 		"AuthenticatedUser": contextGetAuthenticatedUser(r),
// 		"CSRFToken":         nosurf.Token(r),
		"Version": version.Get(),
	}

	return data
}

func (app *Application) newEmailData(r *http.Request) map[string]any {
	data := map[string]any{
	}

	return data
}
