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
{% if cookiecutter.api_only %}{% elif cookiecutter.use_tailwind %}
Tailwind is a standalone binary and is not installed by that task — see the
[Tailwind CLI docs](https://tailwindcss.com/blog/standalone-cli). Until it is
on `PATH` and `task css` has run, `assets/static/css/main.css` does not exist
and every page renders unstyled. `task css:watch` rebuilds it alongside
`task dev`.

Templates name their classes through the `class*` constants in
`internal/ui/templates/views.go`, which is the only file that differs between
a Tailwind project and a hand-written-CSS one.
{% else %}
The stylesheet is inline in `internal/ui/templates/layout.templ`, and class
names come from the `class*` constants in `internal/ui/templates/views.go`.
Nothing is compiled and nothing is fetched, so a page cannot render unstyled.
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

## Logging

`LOG_JSON=false` prints one coloured line per record for local work.
`LOG_JSON=true` emits JSON using [OpenTelemetry][otel] field names — including
for slog's own keys, which are **not** left at their defaults:

| slog default | Emitted as |
|---|---|
| `time` | `timestamp` |
| `level` | `severity_text` |
| `msg` | `body` |
| `error` | `error.message` |

Point log queries and dashboards at those names. The choice lives in one place,
`logging.AccessLogSchema`, and both the application logger and the access log
read it — installing it on only one of them is what produces records that are
half OTEL and half slog. Switching to Elastic Common Schema is a one-word
change there, but it renames every field, so decide before building dashboards
rather than after.

Any record logged with a request context carries `trace_id`, so a line written
deep in a service call ties back to the request that caused it.

Access logs are [httplog][httplog]'s and take their level from the response
status: 5xx error, 4xx warn, everything else info.
Health checks, metrics{% if not cookiecutter.api_only %} and static assets{% endif %} are not logged at all, and a 404
for a route that never matched is logged at info rather than warn, so bot
probing does not turn the dashboard yellow. A handler returning 404 for a
missing resource still warns.

[otel]: https://opentelemetry.io/docs/specs/semconv/general/logs/
[httplog]: https://github.com/go-chi/httplog
{% if not cookiecutter.api_only %}
## HTTP surface

| Path | What |
|---|---|
| `/healthz`, `/version` | Monitoring, also in the OpenAPI spec |
| `/docs`, `/openapi.json` | API reference |
| `/app` | The rendered UI |
| `/app/welcome` | First-run tour of what was scaffolded — delete when it has served its purpose |
| `/app/stream` | SSE endpoint the tour uses to demonstrate server push |
| `/static/*` | Embedded assets |
{% if cookiecutter.use_river -%}
| `{{ '{{' }} RIVER_UI_PATH {{ '}}' }}` | Job dashboard — off unless `RIVER_UI_EMBEDDED=true`, and unauthenticated until you gate it |
{% endif %}
Pages are [templ](https://templ.guide) templates updated in place by
[Datastar](https://data-star.dev). Cross-origin writes are rejected by
`http.CrossOriginProtection`; list any legitimate cross-origin callers in
`TRUSTED_ORIGINS`, comma separated.
{% else %}
## HTTP surface

| Path | What |
|---|---|
| `/healthz`, `/version` | Monitoring |
| `/docs`, `/openapi.json` | API reference |
{% endif %}
## Security headers

Every response carries `Referrer-Policy: no-referrer`, `X-Content-Type-Options:
nosniff`, `X-Frame-Options: DENY` and `Content-Security-Policy:
frame-ancestors 'none'`. There is no fuller CSP by default{% if not cookiecutter.api_only %}: Datastar
evaluates its `data-*` expressions through the `Function` constructor, so any
`script-src` has to permit `'unsafe-eval'`, and a policy that looks stricter
than it is helps nobody. `securityHeaders` in `internal/server/routes.go`
carries a working starting point in a comment{% endif %}.

## API authentication

`ApiKeyAuth` compares an `X-API-Key` header against `X_API_KEY` and is attached
to nothing: a freshly generated project has no endpoint worth guarding. The
scheme is declared in the OpenAPI document so it is ready to reference, and
`registerEndpoints` in `internal/server/routes.go` shows where the gate goes.
Until then no key is required, in production or anywhere else.

## Metrics

Prometheus runtime, process and HTTP metrics are served at `/metrics` on
`METRICS_PORT`, not on `SERVER_PORT`. They sit on a separate listener because
they describe the process and enumerate every route it answers, which is worth
handing an operator and nobody else. The Dockerfile exposes `SERVER_PORT`
alone, so the metrics port is reachable only from wherever the container is
attached.

## Client addresses

`CLIENT_IP_SOURCE` decides what the access log and anything else calling
`middleware.GetClientIP` treats as the caller: `remote` for a process reachable
directly, `header:X-Real-IP` for a proxy that overwrites that header on every
request, or `xff:<cidr>[,<cidr>...]` naming the proxy hops to walk past. Leaving
it on `remote` behind a proxy records the proxy on every line.

## Rate limiting

Not done in this process. Behind a reverse proxy the binary sees the proxy's
address rather than the caller's, so an IP-keyed limiter here would file every
visitor into one bucket and let the first busy client lock out the rest.
Configure it at the edge instead -- Traefik, Caddy and nginx all limit on the
address they terminate.

## Container

```shell
docker build -t {{ cookiecutter.project_slug }} .
```
{% if cookiecutter.database_choice == 'postgres' %}
The image is distroless and has no shell, so the `HEALTHCHECK` uses the
binary's own `healthcheck` subcommand. Supply `DATABASE_URL` at runtime.
{% endif %}
