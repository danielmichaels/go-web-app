package natsio

import (
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"github.com/nats-io/nats.go"
	"log/slog"
)

// Connect establishes a connection to the NATS server
func Connect(cfg *config.Conf, logger *slog.Logger) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("{{ cookiecutter.project_name }}"),
		nats.Timeout(cfg.Nats.Timeout),
		nats.ReconnectWait(time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Warn("NATS connection closed")
		}),
	}

	// todo: authentication must be setup manually

	nc, err := nats.Connect(cfg.Nats.URL, opts...)
	if err != nil {
		return nil, err
	}

	logger.Info("Connected to NATS server", "url", nc.ConnectedUrl())
	return nc, nil
}

{% if cookiecutter.embed_nats %}
// ConnectEmbedded connects to an already-running embedded NATS server.
func ConnectEmbedded(ns interface{ ClientURL() string }, logger *slog.Logger) (*nats.Conn, error) {
	nc, err := nats.Connect(ns.ClientURL(),
		nats.Name("{{ cookiecutter.project_name }}"),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, err
	}
	logger.Info("Connected to embedded NATS server", "url", nc.ConnectedUrl())
	return nc, nil
}
{% endif %}

// Close gracefully closes the NATS connection
func Close(nc *nats.Conn, logger *slog.Logger) {
	if nc != nil {
		nc.Close()
		logger.Info("NATS connection closed")
	}
}
