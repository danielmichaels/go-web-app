// Package templates holds the templ components and the view models they
// render.
//
// View models live in this plain Go file, not in a .templ file: only the
// generated _templ.go is compiled, so a type declared in a .templ would not
// exist until `task templ` had run.
package templates

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
