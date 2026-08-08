// Package templates holds the templ components and the view models they
// render.
//
// View models live in this plain Go file, not in a .templ file: only the
// generated _templ.go is compiled, so a type declared in a .templ would not
// exist until `task templ` had run.
package templates

import "strconv"

// The class constants below are the only place the Tailwind and hand-written
// builds differ. Templates name a class rather than spelling one, so the
// markup is written once and neither build carries a copy of the other's
// names. The hand-written build's stylesheet is inline in Layout.
{% if cookiecutter.use_tailwind -%}
const (
	classShell    = "mx-auto flex max-w-4xl flex-col gap-10 px-5 pt-8 pb-20"
	classTopbar   = "flex items-center justify-between gap-4 border-b border-line pb-4"
	classBrand    = "font-semibold tracking-tight text-ink no-underline"
	classNavLink  = "ml-4 text-sm text-muted no-underline hover:text-accent"
	classSection  = "flex flex-col gap-4"
	classEyebrow  = "font-mono text-xs tracking-widest uppercase text-muted"
	classTitle    = "text-3xl font-semibold tracking-tight text-balance text-ink"
	classSubtitle = "text-lg font-semibold tracking-tight text-ink"
	classLead     = "max-w-xl text-muted"
	classLink     = "text-accent underline underline-offset-2"
	classGrid     = "grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(15rem,1fr))]"
	classCard     = "flex flex-col gap-2 rounded-lg border border-line bg-surface p-4"
	classCardHead = "flex items-baseline justify-between gap-3"
	classCardName = "font-semibold tracking-tight text-ink"
	classCardBody = "text-sm text-muted"
	classCardLink = "text-sm text-accent no-underline hover:underline"
	classTag      = "rounded bg-accent/10 px-1.5 py-0.5 font-mono text-xs whitespace-nowrap text-accent"
	classForm     = "flex flex-wrap items-end gap-2"
	classLabel    = "basis-full text-sm font-medium text-ink"
	classInput    = "min-w-0 flex-1 rounded-md border border-line bg-surface px-2.5 py-2 text-ink"
	// classInputFull is the same control outside a row-direction form. It must
	// not carry flex sizing: in a column-direction parent flex-basis is the
	// height, so the row form's basis would become a tall empty box.
	classInputFull = "w-full rounded-md border border-line bg-surface px-2.5 py-2 text-ink"
	classButton   = "rounded-md bg-accent px-4 py-2 font-medium text-ground hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-50"
	classHint     = "basis-full font-mono text-xs text-muted"
	classHintOver = "basis-full font-mono text-xs text-danger"
	classList     = "divide-y divide-line overflow-hidden rounded-lg border border-line bg-surface text-ink"
	classItem     = "px-3 py-2.5"
	classEmpty    = "rounded-lg border border-dashed border-line p-5 text-center text-muted"
	classFlash    = "rounded-md border-l-[3px] border-accent bg-surface px-3 py-2 text-ink"
	classError    = "rounded-md border-l-[3px] border-danger bg-surface px-3 py-2 text-danger"
	classDot      = "mr-1.5 inline-block size-2 rounded-full bg-accent"
)
{% else -%}
const (
	classShell    = "shell"
	classTopbar   = "topbar"
	classBrand    = "brand"
	classNavLink  = "navlink"
	classSection  = "section"
	classEyebrow  = "eyebrow"
	classTitle    = "title"
	classSubtitle = "subtitle"
	classLead     = "lead"
	classLink     = "link"
	classGrid     = "grid"
	classCard     = "card"
	classCardHead = "card-head"
	classCardName = "card-name"
	classCardBody = "card-body"
	classCardLink = "card-link"
	classTag      = "tag"
	classForm     = "form"
	classLabel    = "label"
	classInput    = "input"
	// classInputFull is the same control outside a row-direction form. It must
	// not carry flex sizing: in a column-direction parent flex-basis is the
	// height, so the row form's basis would become a tall empty box.
	classInputFull = "input full"
	classButton   = "button"
	classHint     = "hint"
	classHintOver = "hint over"
	classList     = "list"
	classItem     = "item"
	classEmpty    = "empty"
	classFlash    = "flash"
	classError    = "error"
	classDot      = "dot"
)
{% endif -%}

// ExampleView is one row of the example list. ID is a string because the
// generated primary key is int32 under pgx and int64 under SQLite.
type ExampleView struct {
	ID   string
	Text string
}

type HomeView struct {
	Title string
	// Flash is a one-shot message surviving exactly one render.
	Flash string
	Items []ExampleView
}

// FeatureView is one card on the welcome tour. Href is optional; a card with
// one links somewhere the reader can go and see the thing working.
type FeatureView struct {
	Name string
	Body string
	Tag  string
	Href string
}

type WelcomeView struct {
	Title    string
	Features []FeatureView
	Items    []ExampleView
}

// PulseView is the payload of the SSE stream: a server clock the browser never
// computes, so a stalled stream is visible rather than merely inferred.
type PulseView struct {
	Clock string
	Ticks int
}

// Updates counts the pushes so far, reading correctly at one as well as many.
func (v PulseView) Updates() string {
	if v.Ticks == 1 {
		return "1 update"
	}
	return strconv.Itoa(v.Ticks) + " updates"
}

// FeatureOpts carries the facts about a running instance that the generated
// feature list cannot know by itself.
type FeatureOpts struct {
{% if cookiecutter.use_river -%}
	// JobsDashboard is where the River dashboard is mounted, empty when it is
	// not being served.
	JobsDashboard string
{% endif -%}
}

// Features describes what this project was generated with. Cards are emitted
// only for what is actually wired up, so the tour never advertises something
// the reader cannot click.
func Features(opts FeatureOpts) []FeatureView {
	features := []FeatureView{
{% if cookiecutter.database_choice == 'postgres' -%}
		{
			Name: "Embedded Postgres",
			Tag:  "database",
			Body: "Development and tests run a real Postgres started by the binary itself. No Docker, nothing installed. A deployment sets DATABASE_URL and the embedded one is skipped.",
		},
{% else -%}
		{
			Name: "SQLite",
			Tag:  "database",
			Body: "A single file, opened by the binary at boot. Migrations are applied in-process, so there is no separate step before the app can serve.",
		},
{% endif -%}
		{
			Name: "OpenAPI documentation",
			Tag:  "huma",
			Body: "Every JSON endpoint is described by a spec generated from the handler signatures, so the two cannot drift apart.",
			Href: "/docs",
		},
		{
			Name: "Health check",
			Tag:  "monitoring",
			Body: "A liveness endpoint the container runtime can poll, kept out of the access log so it does not drown everything else.",
			Href: "/healthz",
		},
		{
			Name: "Sessions and CSRF",
			Tag:  "security",
			Body: "Every page below is already inside a session, and unsafe methods are already checked against a token. The flash message on the home page rides on it.",
		},
		{
			Name: "Structured logging",
			Tag:  "slog",
			Body: "Records carry a trace_id taken from the request, so a line written deep in a service call ties back to the request that caused it.",
		},
{% if cookiecutter.use_river -%}
		{
			Name: "Background jobs",
			Tag:  "river",
			Body: "Jobs are inserted in the same transaction as the write that causes them, so a job can never reference a row that was rolled back.",
			Href: opts.JobsDashboard,
		},
{% endif -%}
{% if cookiecutter.use_nats -%}
		{
{% if cookiecutter.embed_nats -%}
			Name: "Embedded NATS",
			Tag:  "messaging",
			Body: "A NATS server runs inside this process, so messaging works on a laptop with nothing installed and the same code talks to a real cluster in production.",
{% else -%}
			Name: "NATS messaging",
			Tag:  "messaging",
			Body: "A connection and an example subscriber are wired up, ready for work that belongs off the request path.",
{% endif -%}
		},
{% endif -%}
{% if cookiecutter.use_pwa -%}
		{
			Name: "Installable app",
			Tag:  "pwa",
			Body: "A manifest and a service worker ship with the binary, so this can be installed to a home screen and survive a lost connection.",
			Href: "/manifest.json",
		},
{% endif -%}
{% if cookiecutter.use_tailwind -%}
		{
			Name: "Tailwind CSS",
			Tag:  "styling",
			Body: "The stylesheet is compiled from assets/css/input.css by `task css`. Run `task css:watch` alongside `task dev` to rebuild it as you edit.",
		},
{% endif -%}
{% if cookiecutter.ci_choice == 'github' -%}
		{
			Name: "GitHub Actions",
			Tag:  "ci",
			Body: "Lint, test and build run on every push. The Go version comes from go.mod, so upgrading the language is a one-line change.",
		},
{% elif cookiecutter.ci_choice == 'woodpecker' -%}
		{
			Name: "Woodpecker CI",
			Tag:  "ci",
			Body: "Lint, test and build run on every push against a self-hosted runner.",
		},
{% endif -%}
	}

	return features
}
