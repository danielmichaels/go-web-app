package ui

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"{{ cookiecutter.go_module_path.strip() }}/internal/ui/templates"

	datastar "github.com/starfederation/datastar-go/datastar"
)

func (h *Handlers) handleHome(w http.ResponseWriter, r *http.Request) {
	items, err := h.examples(r.Context())
	if err != nil {
		h.serverError(w, r, err, "list examples")
		return
	}

	view := templates.HomeView{
		Title: "{{ cookiecutter.project_name }}",
		Flash: h.Sessions.PopString(r.Context(), flashKey),
		Items: items,
	}
	if err := templates.Home(view).Render(r.Context(), w); err != nil {
		h.serverError(w, r, err, "render home")
	}
}

// handleExampleCreate answers with a Datastar patch rather than a redirect, so
// only the list re-renders. The full page is still reachable on reload because
// handleHome reads the same data.
func (h *Handlers) handleExampleCreate(w http.ResponseWriter, r *http.Request) {
	text := strings.TrimSpace(r.PostFormValue("text"))

	var problem string
	switch {
	case text == "":
		problem = "Text is required."
	case len(text) > 280:
		problem = "Keep it under 280 characters."
	default:
		if err := h.Db.InsertExample(r.Context(), text); err != nil {
			h.serverError(w, r, err, "insert example")
			return
		}
	}

	items, err := h.examples(r.Context())
	if err != nil {
		h.serverError(w, r, err, "list examples")
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElementTempl(templates.ExampleList(items, problem)); err != nil {
		// The response is already streaming, so there is nothing to send but a
		// log line.
		h.Log.ErrorContext(r.Context(), "patch example list", "error", err)
	}
}

// examples converts store rows into the view model. The conversion is not
// ceremony: the generated id is int32 under pgx and int64 under SQLite, so
// templates that touched it directly would only compile for one of them.
func (h *Handlers) examples(ctx context.Context) ([]templates.ExampleView, error) {
	rows, err := h.Db.ExampleSelectAll(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]templates.ExampleView, 0, len(rows))
	for _, row := range rows {
		items = append(items, templates.ExampleView{
			ID:   fmt.Sprint(row.ID),
			Text: row.Text,
		})
	}
	return items, nil
}
