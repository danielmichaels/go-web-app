
package main

import (
	"context"
	"os"
	"os/signal"

	"{{ cookiecutter.go_module_path.strip() }}/internal/commands"
	_ "modernc.org/sqlite"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	ret := commands.Execute(ctx)
	os.Exit(ret)
}
