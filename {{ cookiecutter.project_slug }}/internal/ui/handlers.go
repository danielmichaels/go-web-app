package ui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/ui/templates"

	datastar "github.com/starfederation/datastar-go/datastar"
)

// pulseInterval is how often the welcome tour's stream pushes a new clock.
const pulseInterval = time.Second

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

func (h *Handlers) handleWelcome(w http.ResponseWriter, r *http.Request) {
	items, err := h.examples(r.Context())
	if err != nil {
		h.serverError(w, r, err, "list examples")
		return
	}

	view := templates.WelcomeView{
		Title:    "Welcome to {{ cookiecutter.project_name }}",
		Features: templates.Features(h.featureOpts()),
		Items:    items,
	}
	if err := templates.Welcome(view).Render(r.Context(), w); err != nil {
		h.serverError(w, r, err, "render welcome")
	}
}

// handleStream holds the connection open and pushes a re-rendered fragment on
// every tick. It returns when the browser goes away, which is the only thing
// that ends it.
func (h *Handlers) handleStream(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	ticker := time.NewTicker(pulseInterval)
	defer ticker.Stop()

	for ticks := 1; ; ticks++ {
		view := templates.PulseView{
			Clock: time.Now().Format(time.TimeOnly),
			Ticks: ticks,
		}
		if err := sse.PatchElementTempl(templates.Pulse(view)); err != nil {
			// A closed tab reaches here as a write error, which is ordinary
			// rather than a fault worth logging on every navigation.
			return
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
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

// featureOpts tells the welcome tour what this instance is actually serving,
// as opposed to what it was generated with.
func (h *Handlers) featureOpts() templates.FeatureOpts {
	opts := templates.FeatureOpts{}
{% if cookiecutter.use_river -%}
	if h.Conf.AppConf.RiverUIEnabled {
		opts.JobsDashboard = h.Conf.AppConf.RiverUIPath
	}
{% endif -%}
	return opts
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
