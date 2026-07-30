# {{ cookiecutter.project_name.strip() }}

> {{ cookiecutter.project_description.strip() }}

## Running it

{% if cookiecutter.database_choice == 'postgres' -%}
Nothing needs installing first. Postgres runs inside the process for
development, so there is no database to set up and no credentials to configure.
{% else -%}
Nothing needs installing first. SQLite is a local file under `database/`.
{% endif %}
```shell
task init   # generate code and tidy modules — run once
task dev    # hot-reload server on http://localhost:9898
task test   # no database required
```

Tooling used by those tasks: [task](https://taskfile.dev),
[air](https://github.com/air-verse/air), [sqlc](https://sqlc.dev) and
[goose](https://github.com/pressly/goose). Install them with
`task install_bins`.
{% if cookiecutter.use_tailwind %}
Tailwind is a standalone binary and is not installed by that task — see the
[Tailwind CLI docs](https://tailwindcss.com/blog/standalone-cli).
{% endif %}

## Layout

```
assets/            embedded into the binary: migrations{% if not cookiecutter.api_only %}, static files{% endif %}
  migrations/      goose SQL, applied automatically at boot
  sqlc.yaml        sqlc config, beside the schema it reads
cmd/{{ cookiecutter.cmd_name.strip() }}/         entrypoint and CLI wiring
internal/
  cmd/             kong commands: serve, migrate, healthcheck
  config/          all configuration, decoded from the environment
{% if cookiecutter.database_choice == 'postgres' -%}
  embeddedpg/      runs Postgres as a child process for dev and tests
{% endif -%}
{% if cookiecutter.use_river -%}
  jobs/            River workers and job definitions
{% endif -%}
  logging/         slog setup and the trace_id handler
{% if cookiecutter.use_nats -%}
  natsio/          NATS connection and example subscriber
{% endif -%}
  server/          router, middleware, JSON API handlers
  store/           sqlc output, connection pool, migrations
{% if cookiecutter.database_choice == 'postgres' -%}
  testhelpers/     the shared embedded Postgres used by tests
{% endif -%}
{% if not cookiecutter.api_only -%}
  ui/              templ handlers and templates
{% endif -%}
  version/         build stamping
{% if cookiecutter.database_choice == 'postgres' -%}
scripts/clean-pg.sh   reclaims embedded-Postgres leftovers
{% endif -%}
```

## Database
{% if cookiecutter.database_choice == 'postgres' %}
| | Development / tests | Deployment |
|---|---|---|
| Where Postgres runs | In-process, data in `./.data/postgres` | Wherever `DATABASE_URL` points |
| Selected by | `EMBEDDED_POSTGRES=true` with `DATABASE_URL` empty | `DATABASE_URL` set — always wins |

Migrations are applied by the binary at boot, under an advisory lock so
replicas starting together serialise rather than collide. The `goose` tasks in
the Taskfile are dev conveniences — creating migrations and inspecting status —
not part of deployment.

```shell
task db:migration:create -- add-widgets   # new migration
task db:migration:status                  # what has been applied
task clean                                # drop the dev database and start over
task clean:pg                             # reclaim leftovers from killed test runs
```

Tests share one embedded instance per test binary and isolate with
`TruncateAll`. To use it in a new package:

```go
func TestMain(m *testing.M) { os.Exit(testhelpers.RunTestMain(m)) }

func TestThing(t *testing.T) {
    ctx := context.Background()
    pg := testhelpers.Shared(ctx, t)
    pg.TruncateAll(ctx, t)
    // ...
}
```

> [!IMPORTANT]
> `TruncateAll` lists the application tables by hand. Add every new table to it,
> or rows will leak between tests and show up as an unrelated flake.
{% else %}
SQLite lives at `database/data.db`. Migrations are applied by the binary at
boot; the `goose` tasks are dev conveniences for creating migrations and
inspecting status.

Litestream replicates the file to S3-compatible storage. It has to wrap the
process (`replicate -exec`), which is what the `entrypoint` script is for.

```shell
task db:migration:create -- add-widgets   # new migration
task db:migration:status                  # what has been applied
task clean                                # delete the local database
```
{% endif %}
## Configuration

Everything comes from the environment; `.env` is loaded for local development.
See `.env.example` for the full list. `config.Load` reports every problem at
once, so a misconfigured deployment takes one pass to fix rather than one
restart per missing variable.

Set `APP_ENV=production` in a deployment: it turns the development-friendly
defaults into hard startup errors.
{% if not cookiecutter.api_only %}
## HTTP surface

| Path | What |
|---|---|
| `/healthz`, `/version` | Monitoring, also in the OpenAPI spec |
| `/docs`, `/openapi.json` | API reference |
| `/app` | The rendered UI |
| `/static/*` | Embedded assets |
{% if cookiecutter.use_river -%}
| `{{ '{{' }} RIVER_UI_PATH {{ '}}' }}` | Job dashboard — off unless `RIVER_UI_EMBEDDED=true`, and unauthenticated until you gate it |
{% endif %}
Pages are [templ](https://templ.guide) templates updated in place by
[Datastar](https://data-star.dev). Cross-origin writes are rejected by
`http.CrossOriginProtection`; list any legitimate cross-origin callers in
`TRUSTED_ORIGINS`.
{% else %}
## HTTP surface

| Path | What |
|---|---|
| `/healthz`, `/version` | Monitoring |
| `/docs`, `/openapi.json` | API reference |
{% endif %}
## Container

```shell
docker build -t {{ cookiecutter.project_slug }} .
```
{% if cookiecutter.database_choice == 'postgres' %}
The image is distroless and has no shell, so the `HEALTHCHECK` uses the
binary's own `healthcheck` subcommand. Supply `DATABASE_URL` at runtime.
{% endif %}
