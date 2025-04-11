package cmd

import (
	"{{ cookiecutter.go_module_path.strip() }}/internal/server"
	{% if cookiecutter.database_choice == "postgres" -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
	{% endif -%}
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

	{% if cookiecutter.database_choice == "postgres" -%}
	dbtx := store.New(setup.PgxPool)
	app := server.New(setup.Config, setup.Logger, dbtx, setup.PgxPool)
	{% endif %}
	{% if cookiecutter.database_choice == "sqlite" -%}
	app := server.New(setup.Config, setup.Logger)
	{% endif %}

	err = app.Serve(setup.Ctx)
	if err != nil {
		app.Log.Error("api server error", "error", err, "msg", "failed to start server")
	}
	app.Log.Info("system shutdown")
	return nil
}
