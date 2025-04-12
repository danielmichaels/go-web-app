package cmd

import (
	"{{ cookiecutter.go_module_path.strip() }}/internal/server"
	{% if cookiecutter.database_choice == "postgres" -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
	{% endif -%}
	{% if cookiecutter.use_nats -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/natsio"
	"os"
	{%endif-%}
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

  {% if cookiecutter.use_nats -%}
	natsConn, err := natsio.Connect(setup.Config, setup.Logger)
	if err != nil {
		setup.Logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer natsio.Close(natsConn, setup.Logger)
	// Set up the example subscriber
	exampleSubscriber := natsio.NewExampleSubscriber(natsConn, setup.Logger)
	if err := exampleSubscriber.Subscribe(setup.Ctx); err != nil {
		setup.Logger.Error("Failed to subscribe to example messages", "error", err)
		os.Exit(1)
	}
	defer exampleSubscriber.Unsubscribe()
	{% endif %}

	{% if cookiecutter.database_choice == "postgres" -%}
	dbtx := store.New(setup.PgxPool)
  {% if cookiecutter.use_nats -%}
	app := server.New(setup.Config, setup.Logger, dbtx, setup.PgxPool, natsConn)
  {% else %}
	app := server.New(setup.Config, setup.Logger, dbtx, setup.PgxPool)
	{% endif %}
	{% endif %}
	{% if cookiecutter.database_choice == "sqlite" -%}
  {% if cookiecutter.use_nats -%}
	app := server.New(setup.Config, setup.Logger, natsConn)
  {% else %}
	app := server.New(setup.Config, setup.Logger)
	{% endif %}
	{% endif %}

	err = app.Serve(setup.Ctx)
	if err != nil {
		app.Log.Error("api server error", "error", err, "msg", "failed to start server")
	}
	app.Log.Info("system shutdown")
	return nil
}
