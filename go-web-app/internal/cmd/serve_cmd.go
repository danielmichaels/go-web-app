package cmd

import (
	"github.com/danielmichaels/go-web-app/internal/server"
	"github.com/danielmichaels/go-web-app/internal/store"
	)

const svcAPI = "serve"

type ServeCmd struct {
}

func (s *ServeCmd) validateArgs() error {
	return nil
}

func (s *ServeCmd) Run() error {
	if err := s.validateArgs(); err != nil {
		return err
	}

	setup, err := NewSetup(svcAPI)
	if err != nil {
		return err
	}
	defer setup.Close()

	dbtx := store.New(setup.PgxPool)
	app := server.New(setup.Config, setup.Logger, dbtx, setup.PgxPool)
	
	

	err = app.Serve(setup.Ctx)
	if err != nil {
		app.Log.Error("api server error", "error", err, "msg", "failed to start server")
	}
	app.Log.Info("system shutdown")
	return nil
}
