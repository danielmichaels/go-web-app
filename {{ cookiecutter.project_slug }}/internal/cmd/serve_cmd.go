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
	{% if cookiecutter.use_river -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/jobs"
	{% if not cookiecutter.use_nats -%}
	"os"
	{% endif -%}
	{%endif-%}
)

type ServeCmd struct {
}

func (s *ServeCmd) Run() error {
	app, err := NewApp()
	if err != nil {
		return err
	}
	defer app.Close()

  {% if cookiecutter.use_nats -%}
	{% if cookiecutter.embed_nats -%}
	ns, err := natsio.StartEmbeddedServer(app.Config, app.Logger)
	if err != nil {
		app.Logger.Error("failed to start embedded NATS server", "error", err)
		os.Exit(1)
	}
	defer ns.Shutdown()
	natsConn, err := natsio.ConnectEmbedded(ns, app.Logger)
	{% else -%}
	natsConn, err := natsio.Connect(app.Config, app.Logger)
	{% endif -%}
	if err != nil {
		app.Logger.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer natsio.Close(natsConn, app.Logger)
	exampleSubscriber := natsio.NewExampleSubscriber(natsConn, app.Logger)
	if err := exampleSubscriber.Subscribe(app.Ctx); err != nil {
		app.Logger.Error("Failed to subscribe to example messages", "error", err)
		os.Exit(1)
	}
	defer exampleSubscriber.Unsubscribe()
	{% endif %}

	{% if cookiecutter.use_river -%}
	{% if cookiecutter.database_choice == "postgres" -%}
	jobClient, err := jobs.NewClient(app.Ctx, app.PgxPool, app.Logger)
	{% endif -%}
	{% if cookiecutter.database_choice == "sqlite" -%}
	jobClient, err := jobs.NewClient(app.Ctx, app.Config.Db.DbName, app.Logger)
	{% endif -%}
	if err != nil {
		app.Logger.Error("failed to create job client", "error", err)
		os.Exit(1)
	}
	{% endif %}

	{% if cookiecutter.database_choice == "postgres" -%}
	dbtx := store.New(app.PgxPool)
  {% if cookiecutter.use_nats -%}
  {% if cookiecutter.use_river -%}
	srv := server.New(app.Config, app.Logger, dbtx, app.PgxPool, natsConn, jobClient)
  {% else %}
	srv := server.New(app.Config, app.Logger, dbtx, app.PgxPool, natsConn)
  {% endif %}
  {% else %}
  {% if cookiecutter.use_river -%}
	srv := server.New(app.Config, app.Logger, dbtx, app.PgxPool, jobClient)
  {% else %}
	srv := server.New(app.Config, app.Logger, dbtx, app.PgxPool)
	{% endif %}
	{% endif %}
	{% endif %}
	{% if cookiecutter.database_choice == "sqlite" -%}
  {% if cookiecutter.use_nats -%}
  {% if cookiecutter.use_river -%}
	srv := server.New(app.Config, app.Logger, natsConn, jobClient)
  {% else %}
	srv := server.New(app.Config, app.Logger, natsConn)
  {% endif %}
  {% else %}
  {% if cookiecutter.use_river -%}
	srv := server.New(app.Config, app.Logger, jobClient)
  {% else %}
	srv := server.New(app.Config, app.Logger)
	{% endif %}
	{% endif %}
	{% endif %}

	err = srv.Serve(app.Ctx)
	if err != nil {
		srv.Log.Error("api server error", "error", err, "msg", "failed to start server")
	}
	srv.Log.Info("system shutdown")
	return nil
}
