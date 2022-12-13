package config

import (
	"github.com/joeshaw/envdecode"
	"log"
	"time"
)

type Conf struct {
	Server  serverConf
	Db      dbConf
	Limiter limiter
	Secrets secretKeys
	AppConf appConf
}

type limiter struct {
	Enabled bool          `env:"RATE_LIMIT_ENABLED,default=true"`
	Rps     int           `env:"RATE_LIMIT_RPS,default=10"`
	BackOff time.Duration `env:"RATE_LIMIT_BACKOFF,default=20s"`
}

type dbConf struct {
	DbName                    string        `env:"DATABASE_URL,default=database/data.db"`
	DatabaseConnectionContext time.Duration `env:"DATABASE_CONNECTION_CONTEXT,default=15s"`
	// todo: litestream setup here
}
type serverConf struct {
	Port         int           `env:"SERVER_PORT,default=9898"`
	TimeoutRead  time.Duration `env:"SERVER_TIMEOUT_READ,default=5s"`
	TimeoutWrite time.Duration `env:"SERVER_TIMEOUT_WRITE,default=10s"`
	TimeoutIdle  time.Duration `env:"SERVER_TIMEOUT_IDLE,default=15s"`
}
type appConf struct {
	LogLevel   string `env:"LOG_LEVEL,default=info"`
	LogConcise bool   `env:"LOG_CONCISE,default=true"`
	LogJson    bool   `env:"LOG_JSON,default=false"`
	LogCaller  bool   `env:"LOG_CALLER,default=false"`
}
type secretKeys struct {
	// SMTP
	SmtpHost     string `env:"SMTP_HOST"`
	SmtpPort     int    `env:"SMTP_PORT"`
	SmtpUsername string `env:"SMTP_USERNAME"`
	SmtpPassword string `env:"SMTP_PASSWORD"`
	SmtpSender   string `env:"STMP_SENDER,default={{ cookiecutter.project_slug.strip() }} Accounts<no-reply@{{ cookiecutter.project_slug.strip() }}.com>"`
	// Stripe
	StripeSecretKey           string `env:"STRIPE_SK"`
	StripePublishableKey      string `env:"STRIPE_PK"`
	StripeWebhookSecret       string `env:"STRIPE_WH"`
	// Sentry
	SentryDSN        string `env:"SENTRY_DSN"`
	SentryDebug      bool   `env:"SENTRY_DEBUG,default=false"`
	SentryProduction string `env:"SENTRY_PRODUCTION,default=production"`
}

// AppConfig Setup and install the applications' configuration environment variables
func AppConfig() *Conf {
	var c Conf
	if err := envdecode.StrictDecode(&c); err != nil {
		log.Fatalf("Failed to decode: %s", err)
	}
	return &c
}
