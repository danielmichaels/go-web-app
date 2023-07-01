package commands

import (
	"context"

	"{{ cookiecutter.go_module_path.strip() }}/internal/cmdutils"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/smtp"
	"github.com/spf13/cobra"
)

func EnvOrFlag(flag string, env string) string {
	if flag != "" {
		return flag
	}
	return env
}

func SendMailCmd(ctx context.Context) *cobra.Command {
	var host string
	var port int
	var username string
	var password string
	var from string
	var to string
	cfg := config.AppConfig()
	cmd := &cobra.Command{
		Use:   "sendmail",
		Short: "Send a test email using the in-build smtp server",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := cmdutils.NewLogger("sendmail", cfg)

			body := map[string]any{
				"Name": "Example Person",
			}
			m := smtp.NewMailer(
				EnvOrFlag(host, cfg.Smtp.Host),
				port,
				EnvOrFlag(username, cfg.Smtp.Username),
				EnvOrFlag(password, cfg.Smtp.Password),
				EnvOrFlag(from, cfg.Smtp.Sender),
			)

			err := m.Send(to, body, "example.tmpl")
			if err != nil {
				return err
			}
			logger.Info().Msg("sent email")
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "localhost", "smtp server host domain")
	cmd.Flags().IntVar(&port, "port", 1025, "smtp server host port")
	cmd.Flags().StringVar(&username, "username", "", "smtp server host username")
	cmd.Flags().StringVar(&password, "password", "", "smtp server host password")
	cmd.Flags().StringVar(&from, "from", "no-reply@localhost", "smtp server reply address")
	cmd.Flags().StringVar(&to, "to", "example@localhost", "recipient of the email")
	return cmd
}
