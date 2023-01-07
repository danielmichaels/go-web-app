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

// 	db, err := openDB(cfg)
// 	if err != nil {
// 		logger.Error().Err(err).Msg("unable to connect to database")
// 		os.Exit(1)
// 	}

	app := &server.Application{
		Config:   cfg,
		Logger:   logger,
		RouteDoc: *routeDoc,
// 		Db:       database.New(db),
	}
	err := app.Serve()
	if err != nil {
		app.Logger.Error().Err(err).Msg("server failed to start")
	}
	return nil
}

func openDB(cfg *config.Conf) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", cfg.Db.DbName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Db.DatabaseConnectionContext)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
