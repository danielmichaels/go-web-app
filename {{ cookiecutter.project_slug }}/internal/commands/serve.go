package commands

import (
	"context"
	"{{ cookiecutter.go_module_path.strip() }}/internal/cmdutils"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/repository"
	"{{ cookiecutter.go_module_path.strip() }}/internal/server"
	"{{ cookiecutter.go_module_path.strip() }}/internal/smtp"
	"github.com/spf13/cobra"
)

func ServeCmd(ctx context.Context) *cobra.Command {
	cfg := config.AppConfig()
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "start the webserver",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := cmdutils.NewLogger("server", cfg)

			dbc, err := cmdutils.NewDatabasePool(ctx, cfg)
			if err != nil {
				logger.Fatal().Err(err).Msg("failed to open database")
			}

			defer dbc.Close()
			db := repository.New(dbc)

			mailer := smtp.NewMailer(
				cfg.Smtp.Host,
				cfg.Smtp.Port,
				cfg.Smtp.Username,
				cfg.Smtp.Password,
				cfg.Smtp.Sender,
			)

			app := &server.Application{
				Config: cfg,
				Logger: logger,
				Mailer: mailer,
				Db:     db,
			}

			err = app.Serve()
			if err != nil {
				logger.Fatal().Err(err).Msg("failed to start server")
			}
			return nil
		},
	}
	return cmd
}
