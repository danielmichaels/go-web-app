# {{ cookiecutter.project_name.strip() }}

{{ cookiecutter.project_description.strip() }}

> Replace the placeholder sections below as the project takes shape. What is
> here is what the template knows; the domain is yours to describe.

## What this app does

_One paragraph: who uses it and what problem it solves. Write this first — it
is the context every other decision is judged against._

## Build and run

```shell
task dev     # hot-reload server{% if cookiecutter.database_choice == 'postgres' %}; starts its own Postgres{% endif %}
task test    # no database required
task audit   # lint, align, format
```

Do **not** build a binary to check your work — `task dev` is already running
one, and `go build ./...` is enough to check compilation.
{% if not cookiecutter.api_only %}
`task dev` regenerates templ output on every change. Never edit a `*_templ.go`
file: it is generated and gitignored.
{% endif %}
## Layout

| Path | Contents |
|---|---|
| `assets/` | Embedded into the binary: migrations{% if not cookiecutter.api_only %}, static assets{% endif %} |
| `cmd/{{ cookiecutter.cmd_name.strip() }}/` | Entrypoint, CLI wiring |
| `internal/cmd/` | kong commands: `serve`, `migrate`, `healthcheck` |
| `internal/config/` | All configuration, decoded from the environment |
{% if cookiecutter.database_choice == 'postgres' -%}
| `internal/embeddedpg/` | Runs Postgres as a child process for dev and tests |
{% endif -%}
{% if cookiecutter.use_river -%}
| `internal/jobs/` | River workers and job definitions |
{% endif -%}
| `internal/logging/` | slog setup, `trace_id` handler |
{% if cookiecutter.use_nats -%}
| `internal/natsio/` | NATS connection, example subscriber |
{% endif -%}
| `internal/server/` | Router, middleware, JSON API handlers |
| `internal/store/` | sqlc output, pool, migrations, advisory locks |
{% if cookiecutter.database_choice == 'postgres' -%}
| `internal/testhelpers/` | Shared embedded Postgres for tests |
{% endif -%}
{% if not cookiecutter.api_only -%}
| `internal/ui/` | templ handlers and templates |
{% endif -%}

## Conventions

- **Configuration** is environment-only, decoded once in `config.Load`. Add a
  field with an `env:` tag; never read `os.Getenv` from application code.
- **Errors** use `errors.Is`/`errors.As`, never `err ==`. Wrap with a package
  prefix: `fmt.Errorf("boards: create: %w", err)`.
- **Logging** is `slog`. A record logged with a request context carries
  `trace_id` automatically. Access logs are httplog's job, not yours.
- **Migrations** are goose SQL under `assets/migrations/`, applied by the
  binary at boot. Editing an applied migration is a no-op — add a new one.
- **Queries** are sqlc; write SQL in `internal/store/queries/` and run
  `task sqlc`. Do not hand-write query code.
{% if cookiecutter.database_choice == 'postgres' -%}
- **Transactions and locks**: anything spanning two writes belongs in one
  transaction. Use the advisory-lock helper in `internal/store` rather than
  calling `pg_advisory_lock` directly, and keep every key in that one const
  block — they share a single global namespace per database.
{% endif -%}
{% if not cookiecutter.api_only -%}
- **HTML** is templ; interactivity is Datastar. A handler that updates part of
  a page returns an SSE patch, not a redirect.
{% endif -%}

## Testing

Test-driven where practical: write or update the test first, then implement.
{% if cookiecutter.database_choice == 'postgres' %}
Do **not** mock the database. Tests run against real Postgres via
`internal/testhelpers` — one embedded instance per test binary, no Docker and
nothing installed:

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
> `TruncateAll` lists application tables by hand. **Add every new table to it.**
> A missing one leaks rows between tests and surfaces as a flake in an
> unrelated package.

If tests start leaving Postgres processes behind, run `task clean:pg`. It never
kills anything by default — `FORCE=1` does.
{% endif %}
## Further reading

- `README.md` — running it, HTTP surface, configuration
- `docs/` — architecture notes and runbooks
