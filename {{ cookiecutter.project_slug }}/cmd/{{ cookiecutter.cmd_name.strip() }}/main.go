package main

import (
	"flag"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/server"
	"github.com/go-chi/httplog"
	"log"
)

var routeDoc = flag.Bool("docgen", false, "Generate route documentation using docgen")

func main() {
	err := run()
	if err != nil {
		log.Fatalln("server failed to start:", err)
	}
}

func run() error {
	flag.Parse() // doc gen flag parser
	cfg := config.AppConfig()
	logger := httplog.NewLogger("{{ cookiecutter.project_slug.strip() }}", httplog.Options{
		JSON:     cfg.AppConf.LogJson,
		Concise:  cfg.AppConf.LogConcise,
		LogLevel: cfg.AppConf.LogLevel,
	})
	if cfg.AppConf.LogCaller {
		logger = logger.With().Caller().Logger()
	}

	app := &server.Application{
		Config:   cfg,
		Logger:   logger,
		RouteDoc: *routeDoc,
	}
	err := app.Serve()
	if err != nil {
		app.Logger.Error().Err(err).Msg("server failed to start")
	}
	return nil
}
