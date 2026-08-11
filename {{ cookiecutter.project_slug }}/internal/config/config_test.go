package config_test

import (
	"strings"
	"testing"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
)

// setMinimalEnv supplies just enough for Load to succeed, so each test can
// break exactly one thing and see only that reported.
func setMinimalEnv(t *testing.T) {
	t.Helper()
{% if cookiecutter.database_choice == 'postgres' -%}
	t.Setenv("EMBEDDED_POSTGRES", "true")
{% endif -%}
{% if not cookiecutter.api_only -%}
	t.Setenv("TRUSTED_ORIGINS", "https://trusted.example")
{% endif -%}
}

func TestLoadDefaults(t *testing.T) {
	setMinimalEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() = true, want false: APP_ENV defaults to development")
	}
	if cfg.Server.XApiKey != "" {
		t.Errorf("XApiKey = %q, want empty by default", cfg.Server.XApiKey)
	}
	if cfg.Server.ClientIPSource != "remote" {
		t.Errorf("ClientIPSource = %q, want remote by default", cfg.Server.ClientIPSource)
	}
	if cfg.Server.MetricsPort == cfg.Server.Port {
		t.Errorf("MetricsPort = %d, want a port of its own", cfg.Server.MetricsPort)
	}
}

func TestMetricsPortValidated(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("METRICS_PORT", "70000")

	_, err := config.Load()
	if err == nil {
		t.Fatal("METRICS_PORT=70000 accepted, want rejected")
	}
	if !strings.Contains(err.Error(), "METRICS_PORT") {
		t.Errorf("error does not mention METRICS_PORT:\n%v", err)
	}
}

// One pass should be enough to fix a misconfigured deployment, rather than one
// restart per missing variable.
func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("SERVER_PORT", "70000")
	t.Setenv("METRICS_PORT", "0")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() = nil error, want both problems reported")
	}
	for _, want := range []string{"SERVER_PORT", "METRICS_PORT"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s:\n%v", want, err)
		}
	}
}

// An API key protects nothing until an operation opts into ApiKeyAuth, so a
// deployment that never wires it must not be made to invent a secret.
func TestProductionDoesNotRequireApiKey(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("APP_ENV", "production")
{% if cookiecutter.database_choice == 'postgres' -%}
	t.Setenv("EMBEDDED_POSTGRES", "false")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/app")
{% endif -%}
{% if not cookiecutter.api_only -%}
	t.Setenv("SESSION_COOKIE_SECURE", "true")
{% endif -%}

	if _, err := config.Load(); err != nil {
		t.Fatalf("production rejected with no X_API_KEY set: %v", err)
	}
}
{% if not cookiecutter.api_only %}
// A malformed origin is a typo in an environment variable, so it belongs in
// the same list as every other problem rather than taking the process down
// with a stack trace at boot.
func TestTrustedOriginsValidated(t *testing.T) {
	for _, origin := range []string{
		"example.com",
		"https://x.example/path",
		"https://x.example?a=b",
		"://x.example",
	} {
		t.Run(origin, func(t *testing.T) {
			setMinimalEnv(t)
			t.Setenv("TRUSTED_ORIGINS", origin)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("TRUSTED_ORIGINS=%q accepted, want rejected", origin)
			}
			if !strings.Contains(err.Error(), "TRUSTED_ORIGINS") {
				t.Errorf("error does not mention TRUSTED_ORIGINS:\n%v", err)
			}
		})
	}
}

func TestTrustedOriginsAccepted(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("TRUSTED_ORIGINS", "https://a.example;https://b.example:8443;http://localhost:3000")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("valid origins rejected: %v", err)
	}
	if len(cfg.AppConf.TrustedOrigins) != 3 {
		t.Errorf("TrustedOrigins = %v, want 3 entries", cfg.AppConf.TrustedOrigins)
	}
}

// envdecode splits lists on semicolons while every neighbouring tool uses
// commas, so the likely mistake is named instead of surfacing as one very long
// unparseable origin.
func TestTrustedOriginsNamesTheSeparator(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("TRUSTED_ORIGINS", "https://a.example,https://b.example")

	_, err := config.Load()
	if err == nil {
		t.Fatal("comma-separated origins accepted, want rejected")
	}
	if !strings.Contains(err.Error(), "';'") {
		t.Errorf("error does not point at the separator:\n%v", err)
	}
}
{% endif -%}

// chi builds the XFF middleware with netip.MustParsePrefix and panics at boot
// on a bad prefix, so an unusable source has to be caught during Load.
func TestClientIPSourceRejected(t *testing.T) {
	for _, source := range []string{
		"bogus",
		"remote:x",
		"header:",
		"xff:",
		"xff:not-a-cidr",
		"xff:10.0.0.0/8,nonsense",
	} {
		t.Run(source, func(t *testing.T) {
			setMinimalEnv(t)
			t.Setenv("CLIENT_IP_SOURCE", source)

			if _, err := config.Load(); err == nil {
				t.Errorf("CLIENT_IP_SOURCE=%q accepted, want rejected", source)
			}
		})
	}
}

func TestClientIPSourceAccepted(t *testing.T) {
	for _, source := range []string{
		"remote",
		"header:X-Real-IP",
		"xff:10.0.0.0/8",
		"xff:10.0.0.0/8,172.16.0.0/12",
		"xff:2600:9000::/28",
	} {
		t.Run(source, func(t *testing.T) {
			setMinimalEnv(t)
			t.Setenv("CLIENT_IP_SOURCE", source)

			if _, err := config.Load(); err != nil {
				t.Errorf("CLIENT_IP_SOURCE=%q rejected: %v", source, err)
			}
		})
	}
}
{% if cookiecutter.use_river and not cookiecutter.api_only %}
// chi panics on overlapping mounts, so a bad RIVER_UI_PATH would kill the
// process at boot. Load rejects it with an explanation instead.
func TestRiverUIPathRejected(t *testing.T) {
	for _, path := range []string{"/", "/app", "/app/jobs", "/static", "/riverui/", "riverui"} {
		t.Run(path, func(t *testing.T) {
			setMinimalEnv(t)
			t.Setenv("RIVER_UI_EMBEDDED", "true")
			t.Setenv("RIVER_UI_PATH", path)

			if _, err := config.Load(); err == nil {
				t.Errorf("RIVER_UI_PATH=%q accepted, want rejected", path)
			}
		})
	}
}

// Nothing mounts when the dashboard is off, so the path cannot collide.
func TestRiverUIPathUncheckedWhenDisabled(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("RIVER_UI_EMBEDDED", "false")
	t.Setenv("RIVER_UI_PATH", "/app")

	if _, err := config.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
}

func TestRiverUIDefaultsAreUsable(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("RIVER_UI_EMBEDDED", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := cfg.AppConf.RiverUIPath; got != "/riverui" {
		t.Errorf("RiverUIPath = %q, want /riverui", got)
	}
}
{% endif -%}
