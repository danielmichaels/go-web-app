{% if cookiecutter.use_nats and cookiecutter.embed_nats %}
package natsio

import (
	"fmt"
	"log/slog"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	natsserver "github.com/nats-io/nats-server/v2/server"
)

func StartEmbeddedServer(cfg *config.Conf, logger *slog.Logger) (*natsserver.Server, error) {
	port := cfg.Nats.Port
	if port == 0 {
		port = -1
	}
	opts := &natsserver.Options{
		Host:      "127.0.0.1",
		Port:      port,
		JetStream: true,
		StoreDir:  cfg.Nats.StoreDir,
		NoSigs:    true,
	}

	ns, err := natsserver.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("creating embedded NATS server: %w", err)
	}

	ns.SetLoggerV2(
		&natsLogger{logger: logger},
		false, false, false,
	)

	go ns.Start()

	if !ns.ReadyForConnections(5 * time.Second) {
		return nil, fmt.Errorf("embedded NATS server not ready within timeout")
	}

	logger.Info("embedded NATS server started", "url", ns.ClientURL())
	return ns, nil
}

// natsLogger adapts slog to the nats-server Logger interface.
type natsLogger struct {
	logger *slog.Logger
}

func (l *natsLogger) Noticef(format string, v ...any) {
	l.logger.Info(fmt.Sprintf(format, v...))
}
func (l *natsLogger) Warnf(format string, v ...any) {
	l.logger.Warn(fmt.Sprintf(format, v...))
}
func (l *natsLogger) Fatalf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}
func (l *natsLogger) Errorf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}
func (l *natsLogger) Debugf(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}
func (l *natsLogger) Tracef(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}
{% endif %}
