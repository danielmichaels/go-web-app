package cmd

import (
	"fmt"

{% if cookiecutter.use_river -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/jobs"
{% endif -%}
{% if cookiecutter.use_nats -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/natsio"
{% endif -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/server"
)

type ServeCmd struct {
}

func (s *ServeCmd) Run() error {
	app, err := NewApp()
	if err != nil {
		return err
	}
	defer app.Close()

	deps := server.Deps{
		Conf: app.Config,
		Log:  app.Logger,
		Db:   app.Store,
{% if cookiecutter.database_choice == 'postgres' -%}
		PgxPool: app.PgxPool,
{% endif -%}
	}

{% if cookiecutter.use_nats -%}
{% if cookiecutter.embed_nats -%}
	ns, err := natsio.StartEmbeddedServer(app.Config, app.Logger)
	if err != nil {
		return fmt.Errorf("start embedded NATS: %w", err)
	}
	defer ns.Shutdown()
	natsConn, err := natsio.ConnectEmbedded(ns, app.Logger)
{% else -%}
	natsConn, err := natsio.Connect(app.Config, app.Logger)
{% endif -%}
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	defer natsio.Close(natsConn, app.Logger)

	exampleSubscriber := natsio.NewExampleSubscriber(natsConn, app.Logger)
	if err := exampleSubscriber.Subscribe(app.Ctx); err != nil {
		return fmt.Errorf("subscribe to example messages: %w", err)
	}
	// The process is on its way out and the connection close above tears the
	// subscription down regardless, so a failed unsubscribe has nothing left
	// to act on.
	//nolint:errcheck
	defer exampleSubscriber.Unsubscribe()

	deps.Nats = natsConn
{% endif -%}

{% if cookiecutter.use_river -%}
{% if cookiecutter.database_choice == 'postgres' -%}
	jobClient, err := jobs.NewClient(app.Ctx, app.PgxPool, app.Config, app.Logger)
{% else -%}
	jobClient, err := jobs.NewClient(app.Ctx, app.Config.Db.DbName, app.Config, app.Logger)
{% endif -%}
	if err != nil {
		return fmt.Errorf("create job client: %w", err)
	}
	deps.Jobs = jobClient
{% endif -%}

	srv := server.New(deps)

	if err := srv.Start(app.Ctx); err != nil {
		return fmt.Errorf("start background workers: %w", err)
	}
	defer func() {
		if err := srv.Stop(app.Ctx); err != nil {
			app.Logger.Error("stopping background workers", "error", err)
		}
	}()

	if err := srv.Serve(app.Ctx); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
