package config

import (
	"github.com/joeshaw/envdecode"
	"log"
	"log/slog"
	"time"
)

type Conf struct {
	Server  serverConf
	Db      dbConf
	Limiter limiter
	AppConf appConf
	{% if cookiecutter.use_nats -%}
	Nats natsConf
	{% endif -%}
}

type limiter struct {
	Enabled bool          `env:"RATE_LIMIT_ENABLED,default=true"`
	Rps     int           `env:"RATE_LIMIT_RPS,default=10"`
	BackOff time.Duration `env:"RATE_LIMIT_BACKOFF,default=20s"`
}

{% if cookiecutter.use_nats -%}
type natsConf struct {
	URL string `env:"NATS_URL,default=nats://localhost:4222"`
	Timeout  time.Duration `env:"NATS_TIMEOUT,default=10s"`
}
{% endif -%}

type dbConf struct {
	Host     string `env:"POSTGRES_HOST,default=localhost"`
	Db       string `env:"POSTGRES_DB,default=db"`
	User     string `env:"POSTGRES_USER,default=dbuser"`
	Password string `env:"POSTGRES_PASSWORD,default=dbuser"`
	// PG SSL MODES: allow, disable
	SSLMode  string `env:"POSTGRES_SSL_MODE,default=disable"`
	Port     int    `env:"POSTGRES_PORT,default=5432"`
	MaxConns int    `env:"POSTGRES_MAX_CONNS,default=16"`

	// sqliteonly
	DbName                    string        `env:"DATABASE_URL,default=database/data.db"`
	DatabaseConnectionContext time.Duration `env:"DATABASE_CONNECTION_CONTEXT,default=15s"`
}
type serverConf struct {
	// temporary XApiKey for development
	XApiKey string `env:"X_API_KEY,default=changeme"`

	Port         int           `env:"SERVER_PORT,default=9898"`
	TimeoutRead  time.Duration `env:"SERVER_TIMEOUT_READ,default=5s"`
	TimeoutWrite time.Duration `env:"SERVER_TIMEOUT_WRITE,default=10s"`
	TimeoutIdle  time.Duration `env:"SERVER_TIMEOUT_IDLE,default=15s"`
}
type appConf struct {
	LogLevel           slog.Level `env:"LOG_LEVEL,default=info"`
	LogJson            bool       `env:"LOG_JSON,default=false"`
	LogConcise         bool       `env:"LOG_CONCISE,default=false"`
	LogResponseHeaders bool       `env:"LOG_RESPONSE_HEADERS,default=false"`
	LogRequestHeaders  bool       `env:"LOG_REQUEST_HEADERS,default=true"`
}

// AppConfig Setup and install the applications' configuration environment variables
func AppConfig() *Conf {
	var c Conf
	if err := envdecode.StrictDecode(&c); err != nil {
		log.Fatalf("Failed to decode: %s", err)
	}
	return &c
}
