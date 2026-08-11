package config

import (
{% if not cookiecutter.api_only -%}
	"net/http"
{% endif -%}
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/joeshaw/envdecode"
)

// EnvProduction is the APP_ENV value that turns the development-only defaults
// below into hard configuration errors.
const EnvProduction = "production"

type Conf struct {
	Server  serverConf
	Db      dbConf
	AppConf appConf
{% if not cookiecutter.api_only -%}
	Session sessionConf
{% endif -%}
{% if cookiecutter.use_nats -%}
	Nats natsConf
{% endif -%}
}

{% if cookiecutter.use_nats -%}
type natsConf struct {
	URL     string        `env:"NATS_URL,default=nats://localhost:4222"`
	Timeout time.Duration `env:"NATS_TIMEOUT,default=10s"`
{% if cookiecutter.embed_nats -%}
	StoreDir string `env:"NATS_STORE_DIR,default=data/jetstream"`
	Port     int    `env:"NATS_EMBED_PORT,default=0"`
{% endif -%}
}
{% endif -%}

{% if cookiecutter.database_choice == 'postgres' -%}
type dbConf struct {
	// URL is the only connection setting. Leave it empty in development and
	// the embedded instance fills it in at boot; set it in a deployment to
	// point at a real server.
	URL      string `env:"DATABASE_URL,default="`
	MaxConns int    `env:"DATABASE_MAX_CONNS,default=10"`

	Embedded bool `env:"EMBEDDED_POSTGRES,default=false"`
	// EmbeddedDir deliberately sits outside tmp/: that is air's scratch
	// directory, and a rebuild there would take the dev database with it.
	EmbeddedDir  string `env:"EMBEDDED_POSTGRES_DIR,default=./.data/postgres"`
	EmbeddedPort int    `env:"EMBEDDED_POSTGRES_PORT,default=5433"`
}
{% else -%}
type dbConf struct {
	DbName                    string        `env:"DATABASE_URL,default=database/data.db"`
	DatabaseConnectionContext time.Duration `env:"DATABASE_CONNECTION_CONTEXT,default=15s"`
}
{% endif -%}

type serverConf struct {
	XApiKey      string        `env:"X_API_KEY,default="`
	Port         int           `env:"SERVER_PORT,default=9898"`
	// MetricsPort carries /metrics on a listener of its own so the interface
	// serving it can be kept off the public network. Nothing publishes it:
	// the Dockerfile exposes SERVER_PORT alone.
	MetricsPort  int           `env:"METRICS_PORT,default=9899"`
	TimeoutRead  time.Duration `env:"SERVER_TIMEOUT_READ,default=5s"`
	TimeoutWrite time.Duration `env:"SERVER_TIMEOUT_WRITE,default=10s"`
	TimeoutIdle  time.Duration `env:"SERVER_TIMEOUT_IDLE,default=15s"`
	// ClientIPSource decides where a caller's address is read from: remote,
	// header:<Name>, or xff:<cidr>[,<cidr>...]. It has to match the
	// deployment. A header is only as trustworthy as the proxy that
	// overwrites it on every request, and remote behind any proxy yields the
	// proxy's own address rather than the caller's.
	ClientIPSource string `env:"CLIENT_IP_SOURCE,default=remote"`
}

{% if not cookiecutter.api_only -%}
// sessionConf configures server-side sessions. There is no secret to set:
// scs stores session data in Postgres and the cookie carries only an opaque
// token.
type sessionConf struct {
	Lifetime time.Duration `env:"SESSION_LIFETIME,default=168h"`
	// Secure must be true anywhere the site is served over HTTPS.
	Secure bool `env:"SESSION_COOKIE_SECURE,default=false"`
}

// originList is a comma-separated list of origins.
//
// It parses itself because envdecode splits a plain []string on semicolons,
// which is not the separator anyone reaches for: a second origin written the
// expected way became part of the first and was rejected as one unparseable
// value. envdecode prefers a TextUnmarshaler over its own slice handling, so
// declaring one is enough to take the decision back.
type originList []string

func (o *originList) UnmarshalText(text []byte) error {
	*o = splitList(string(text))
	return nil
}

// NewCrossOriginProtection builds the stdlib CSRF check, reporting the entries
// it refused rather than failing on the first. Load and the server both build
// it from here, so neither can accept an origin the other rejects.
func NewCrossOriginProtection(origins []string) (*http.CrossOriginProtection, []string) {
	p := http.NewCrossOriginProtection()
	var problems []string
	for _, origin := range origins {
		if err := p.AddTrustedOrigin(origin); err != nil {
			problems = append(problems, fmt.Sprintf("TRUSTED_ORIGINS: %v", err))
		}
	}
	return p, problems
}
{% endif -%}

type appConf struct {
	Env                string     `env:"APP_ENV,default=development"`
	LogLevel           slog.Level `env:"LOG_LEVEL,default=info"`
	LogJson            bool       `env:"LOG_JSON,default=false"`
	LogConcise         bool       `env:"LOG_CONCISE,default=false"`
	LogResponseHeaders bool       `env:"LOG_RESPONSE_HEADERS,default=false"`
	LogRequestHeaders  bool       `env:"LOG_REQUEST_HEADERS,default=true"`
{% if not cookiecutter.api_only -%}
	// TrustedOrigins are the cross-origin sites allowed to submit to this app,
	// as scheme://host (e.g. https://app.example.com).
	TrustedOrigins originList `env:"TRUSTED_ORIGINS"`
{% endif -%}
{% if cookiecutter.use_river and not cookiecutter.api_only -%}
	// RiverUIEnabled mounts the job dashboard inside this binary. It ships off
	// because the dashboard can cancel and retry jobs and nothing in a freshly
	// generated project authorises anything — see the mount site in
	// internal/server/routes.go for where the gate goes.
	RiverUIEnabled bool `env:"RIVER_UI_EMBEDDED,default=false"`
	// RiverUIPath is checked against the paths Routes already mounts.
	RiverUIPath string `env:"RIVER_UI_PATH,default=/riverui"`
{% endif -%}
}

// IsProduction reports whether this process is configured as a deployment
// rather than a developer's machine.
func (c *Conf) IsProduction() bool { return c.AppConf.Env == EnvProduction }

{% if cookiecutter.database_choice == 'postgres' -%}
// ShouldStartEmbedded reports whether this process owns its own Postgres. A
// supplied DATABASE_URL always wins, so a deployment never accidentally boots
// a throwaway database alongside the real one.
func ShouldStartEmbedded(c *Conf) bool {
	return c.Db.Embedded && c.Db.URL == ""
}
{% endif -%}

{% if cookiecutter.use_river and not cookiecutter.api_only -%}
// mountedPrefixes are the paths Routes already owns. chi panics when two
// mounts overlap, so an unlucky RIVER_UI_PATH would take the whole process
// down at boot rather than 404 on one route.
var mountedPrefixes = []string{"/app", "/static", "/docs", "/healthz", "/version", "/openapi.json"}

// riverUIPathProblem returns the empty string when the path is usable.
func riverUIPathProblem(path string) string {
	switch {
	case !strings.HasPrefix(path, "/"):
		return "RIVER_UI_PATH must start with /"
	case path == "/":
		return "RIVER_UI_PATH must not be /: the dashboard would answer every route"
	case strings.HasSuffix(path, "/"):
		return "RIVER_UI_PATH must not end with /"
	}
	for _, prefix := range mountedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return fmt.Sprintf("RIVER_UI_PATH must not sit under %s, which is already mounted", prefix)
		}
	}
	return ""
}

{% endif -%}
// splitList splits a comma-separated environment value, dropping surrounding
// space and empty entries.
func splitList(s string) []string {
	var out []string
	for _, entry := range strings.Split(s, ",") {
		if v := strings.TrimSpace(entry); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// CLIENT_IP_SOURCE modes, spelled as they are written in the environment.
const (
	ClientIPRemote = "remote"
	ClientIPHeader = "header"
	ClientIPXFF    = "xff"
)

// ClientIPSource is CLIENT_IP_SOURCE parsed into the argument its middleware
// needs. Only Mode is always set; Header and Prefixes belong to one mode each.
type ClientIPSource struct {
	Mode     string
	Header   string
	Prefixes []string
}

// ParseClientIPSource parses CLIENT_IP_SOURCE. Load reports the error beside
// every other configuration problem and the router wires the result, so the
// grammar is defined once and the two cannot drift.
//
// Prefixes are parsed here rather than left to chi, whose XFF middleware
// builds them with netip.MustParsePrefix and would take the process down at
// boot instead of reporting the typo.
func ParseClientIPSource(source string) (ClientIPSource, error) {
	mode, arg, _ := strings.Cut(source, ":")
	switch mode {
	case ClientIPRemote:
		if arg != "" {
			return ClientIPSource{}, errors.New("CLIENT_IP_SOURCE=remote takes no argument")
		}
		return ClientIPSource{Mode: mode}, nil
	case ClientIPHeader:
		if arg == "" {
			return ClientIPSource{}, errors.New(
				"CLIENT_IP_SOURCE=header needs a header name, e.g. header:X-Real-IP",
			)
		}
		return ClientIPSource{Mode: mode, Header: arg}, nil
	case ClientIPXFF:
		prefixes := splitList(arg)
		if len(prefixes) == 0 {
			return ClientIPSource{}, errors.New(
				"CLIENT_IP_SOURCE=xff needs at least one trusted CIDR, e.g. xff:10.0.0.0/8",
			)
		}
		for _, cidr := range prefixes {
			if _, err := netip.ParsePrefix(cidr); err != nil {
				return ClientIPSource{}, fmt.Errorf(
					"CLIENT_IP_SOURCE has an unparseable CIDR %q",
					cidr,
				)
			}
		}
		return ClientIPSource{Mode: mode, Prefixes: prefixes}, nil
	}
	return ClientIPSource{}, fmt.Errorf(
		"CLIENT_IP_SOURCE %q must be remote, header:<Name>, or xff:<cidr>[,<cidr>...]",
		source,
	)
}

// Load reads configuration from the environment.
//
// Every problem is reported at once rather than one per restart: a
// misconfigured deployment should need a single pass to fix, not one
// deploy per missing variable.
func Load() (*Conf, error) {
	var c Conf
	if err := envdecode.StrictDecode(&c); err != nil {
		return nil, fmt.Errorf("config: decode: %w", err)
	}

	var problems []string

{% if cookiecutter.database_choice == 'postgres' -%}
	if c.Db.URL == "" && !c.Db.Embedded {
		problems = append(
			problems,
			"DATABASE_URL must be set, or EMBEDDED_POSTGRES=true to run one in-process",
		)
	}
	if c.Db.MaxConns < 1 {
		problems = append(problems, "DATABASE_MAX_CONNS must be at least 1")
	}
{% endif -%}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		problems = append(problems, "SERVER_PORT must be between 1 and 65535")
	}
	if _, err := ParseClientIPSource(c.Server.ClientIPSource); err != nil {
		problems = append(problems, err.Error())
	}
	if c.Server.MetricsPort < 1 || c.Server.MetricsPort > 65535 {
		problems = append(problems, "METRICS_PORT must be between 1 and 65535")
	}
	if c.Server.MetricsPort == c.Server.Port {
		problems = append(
			problems,
			"METRICS_PORT must differ from SERVER_PORT; both listeners cannot share one port",
		)
	}
{% if cookiecutter.use_river and not cookiecutter.api_only -%}
	if c.AppConf.RiverUIEnabled {
		if problem := riverUIPathProblem(c.AppConf.RiverUIPath); problem != "" {
			problems = append(problems, problem)
		}
	}
{% endif -%}

{% if not cookiecutter.api_only -%}
	// Built here as well as where it is used, so a typo is reported beside
	// every other problem instead of ending the process with a stack trace
	// before anything has started.
	_, originProblems := NewCrossOriginProtection(c.AppConf.TrustedOrigins)
	problems = append(problems, originProblems...)
{% endif -%}

{# Each check guards itself. One shared `if c.IsProduction()` would render
   empty for a SQLite api_only build -- staticcheck SA9003 -- and its Jinja
   condition would be a second copy of the guards below, silently dropping any
   later check that did not happen to match it. #}
{% if cookiecutter.database_choice == 'postgres' -%}
	if c.IsProduction() && c.Db.Embedded {
		problems = append(
			problems,
			"EMBEDDED_POSTGRES must be false in production; supply DATABASE_URL instead",
		)
	}
{% endif -%}
{% if not cookiecutter.api_only -%}
	if c.IsProduction() && len(c.AppConf.TrustedOrigins) == 0 {
		problems = append(problems, "TRUSTED_ORIGINS must list at least one origin in production")
	}
	if c.IsProduction() && !c.Session.Secure {
		problems = append(problems, "SESSION_COOKIE_SECURE must be true in production")
	}
{% endif -%}

	if len(problems) > 0 {
		return nil, fmt.Errorf("config:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return &c, nil
}
