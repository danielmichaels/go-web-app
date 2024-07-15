package cmd

import (
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"github.com/go-chi/httplog"
	"github.com/rs/zerolog"
)

type Globals struct {
}

func NewLogger(name string, cfg *config.Conf) *zerolog.Logger {
	logger := httplog.NewLogger(name, httplog.Options{
		JSON:     cfg.AppConf.LogJson,
		Concise:  cfg.AppConf.LogConcise,
		LogLevel: cfg.AppConf.LogLevel,
	})
	if cfg.AppConf.LogCaller {
		logger = logger.With().Caller().Logger()
	}
	return &logger
}

