package server_test

import (
	"context"
	"net/http"
	"testing"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/server"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

// guardedAPI puts one operation behind ApiKeyAuth, so a request either reaches
// the handler or it does not. No database: the middleware reads only config.
func guardedAPI(t *testing.T, apiKey string) humatest.TestAPI {
	t.Helper()
{% if cookiecutter.database_choice == 'postgres' -%}
	t.Setenv("EMBEDDED_POSTGRES", "true")
{% endif -%}
{% if not cookiecutter.api_only -%}
	t.Setenv("TRUSTED_ORIGINS", "https://trusted.example")
{% endif -%}
	t.Setenv("X_API_KEY", apiKey)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	app := &server.App{Deps: server.Deps{Conf: cfg}}
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	huma.Register(api, huma.Operation{
		OperationID:   "guarded",
		Method:        http.MethodGet,
		Path:          "/guarded",
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{app.ApiKeyAuth(api)},
	}, func(context.Context, *struct{}) (*struct{}, error) { return nil, nil })

	return api
}

// An unset key authorises nothing. Comparing an absent header against an empty
// setting would otherwise admit every request that left the header off.
func TestApiKeyAuthFailsClosedWhenUnset(t *testing.T) {
	api := guardedAPI(t, "")

	if res := api.Get("/guarded"); res.Code != http.StatusUnauthorized {
		t.Errorf("no header: status = %d, want 401", res.Code)
	}
	if res := api.Get("/guarded", "X-API-Key: anything"); res.Code != http.StatusUnauthorized {
		t.Errorf("with header: status = %d, want 401", res.Code)
	}
}

func TestApiKeyAuthRejectsWrongKey(t *testing.T) {
	api := guardedAPI(t, "correct-horse")

	if res := api.Get("/guarded", "X-API-Key: wrong"); res.Code != http.StatusUnauthorized {
		t.Errorf("wrong key: status = %d, want 401", res.Code)
	}
	if res := api.Get("/guarded"); res.Code != http.StatusUnauthorized {
		t.Errorf("missing header: status = %d, want 401", res.Code)
	}
}

func TestApiKeyAuthAcceptsCorrectKey(t *testing.T) {
	api := guardedAPI(t, "correct-horse")

	if res := api.Get("/guarded", "X-API-Key: correct-horse"); res.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", res.Code)
	}
}
