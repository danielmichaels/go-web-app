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
}

// One pass should be enough to fix a misconfigured deployment, rather than one
// restart per missing variable.
func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SERVER_PORT", "70000")
	// X_API_KEY is left at its default, which production rejects.

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() = nil error, want the production problems reported")
	}
	for _, want := range []string{"SERVER_PORT", "X_API_KEY"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %s:\n%v", want, err)
		}
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
