package cmd

import (
	"context"
	"fmt"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/webserver"
)

type ServeCmd struct{}

func (s *ServeCmd) Run(g *Globals) error {
	fmt.Println("Serve called")
	cfg := config.AppConfig()
	logger := NewLogger("server", cfg)

	//db, err := NewDatabasePool(ctx, cfg)
	//if err != nil {
	//	logger.Fatal().Err(err).Msg("failed to open database. exiting.")
	//}
	//defer db.Close()
	//err = db.Ping(ctx)
	//if err != nil {
	//	logger.Fatal().Err(err).Msg("failed to ping database. exiting.")
	//}
	//natsConn, err := natsio.Connect(cfg.Server.NatsURI)
	//if err != nil {
	//	return err
	//}
	//natsEconn, err := natsio.EConnect(natsConn)
	//if err != nil {
	//	return err
	//}
	//dbtx := repository.New(db)
	//nc := natsio.NewNats(natsConn, natsEconn, logger, dbtx)
	app := &webserver.Application{
		Config: cfg,
		Logger: logger,
// 		Db:       dbtx,
		//Nats:     nc,
	}

	//logger.Info().Str("nats", app.Nats.Conn.ConnectedUrlRedacted()).Msg("connected to NATS")
	//err = app.Nats.InitSubscribers()
	//if err != nil {
	//	logger.Fatal().Err(err).Msg("error: failed to initialise subscribers")
	//}
	err := app.Serve(context.Background())
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start server")
	}
	// Set up the interrupt handler to drain, so we don't miss
	// requests when scaling down.
	app.Logger.Info().Msg("Draining NATS")
	//err = app.Nats.EncConn.Drain()
	//if err != nil {
	//	logger.Error().Err(err).Msg("error: failed to drain messages")
	//}
	logger.Info().Msg("system shutdown")

	return nil
}
