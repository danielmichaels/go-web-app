module {{ cookiecutter.go_module_path.strip('/') }}

go 1.26

// Direct dependencies only. `task init` runs `go mod tidy`, which resolves the
// indirect set and writes go.sum.
require (
	github.com/SladkyCitron/slogcolor v1.9.0
	github.com/alecthomas/kong v1.16.0
	github.com/danielgtaylor/huma/v2 v2.39.1
	github.com/felixge/httpsnoop v1.0.4
	github.com/go-chi/chi/v5 v5.3.1
	github.com/go-chi/httplog/v3 v3.4.0
	github.com/joeshaw/envdecode v0.0.0-20200121155833-099f1fc765bd
	github.com/pressly/goose/v3 v3.27.3
	github.com/prometheus/client_golang v1.23.1
	golang.org/x/sync v0.22.0
{%- if cookiecutter.database_choice == 'postgres' %}
	github.com/fergusstrange/embedded-postgres v1.34.0
	github.com/jackc/pgx/v5 v5.10.0
{%- else %}
	modernc.org/sqlite v1.55.0
{%- endif %}
{%- if not cookiecutter.api_only %}
	github.com/a-h/templ v0.3.1020
	github.com/alexedwards/scs/v2 v2.9.0
	github.com/starfederation/datastar-go v1.2.2
{%- if cookiecutter.database_choice == 'postgres' %}
	github.com/alexedwards/scs/pgxstore v0.0.0-20251002162104-209de6e426de
{%- endif %}
{%- endif %}
{%- if cookiecutter.use_nats %}
	github.com/nats-io/nats.go v1.52.0
{%- if cookiecutter.embed_nats %}
	github.com/nats-io/nats-server/v2 v2.14.3
{%- endif %}
{%- endif %}
{%- if cookiecutter.use_river %}
	github.com/riverqueue/river v0.41.1
{%- if cookiecutter.database_choice == 'postgres' %}
	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.41.1
{%- else %}
	github.com/riverqueue/river/riverdriver/riversqlite v0.41.1
{%- endif %}
{%- if not cookiecutter.api_only %}
	riverqueue.com/riverui v0.16.0
{%- endif %}
{%- endif %}
)
{%- if not cookiecutter.api_only %}

// templ generates the Go code behind every .templ file. Pinning the generator
// here keeps `go tool templ generate` on the same version as the runtime
// library the generated code compiles against.
tool github.com/a-h/templ/cmd/templ
{%- endif %}
