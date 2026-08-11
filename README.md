# go-web-app

> A [cookiecutter](https://cookiecutter.readthedocs.io/en/stable/) for quickly scaffolding Go web applications

An opinionated Go web app template. Batteries included, production-ready defaults, optional components selected at generation time.

The headline property: **a generated project needs nothing installed to run.**
`git clone && task dev` and `go test ./...` work on a bare laptop — no Postgres,
no Docker, no credentials. Postgres runs embedded inside the process for
development and tests; a deployment supplies `DATABASE_URL` and the embedded
instance is skipped entirely.

## Stack

| Component | Library |
|---|---|
| HTTP router | [chi](https://github.com/go-chi/chi) |
| API / OpenAPI | [huma](https://huma.rocks), with its built-in [Scalar](https://scalar.com) reference at `/docs` |
| HTML | [templ](https://templ.guide) |
| Interactivity | [Datastar](https://data-star.dev) |
| Sessions | [scs](https://github.com/alexedwards/scs) (+ `pgxstore`) |
| CLI | [kong](https://github.com/alecthomas/kong) |
| Logging | [slog](https://pkg.go.dev/log/slog) + [slogcolor](https://github.com/SladkyCitron/slogcolor) + [httplog](https://github.com/go-chi/httplog) |
| Dev/test database | [embedded-postgres](https://github.com/fergusstrange/embedded-postgres) |
| Migrations | [goose](https://github.com/pressly/goose) (applied in-process at boot) |
| Query gen | [sqlc](https://sqlc.dev) |
| Jobs | [River](https://riverqueue.com) |
| Metrics | [prometheus/client_golang](https://github.com/prometheus/client_golang), on a listener of its own |
| Hot reload | [air](https://github.com/air-verse/air) |
| Task runner | [task](https://taskfile.dev) |

## Usage

> [!NOTE]
> Requires [uvx](https://docs.astral.sh/uv/guides/tools/) or `pip install cookiecutter`

```shell
uvx cookiecutter https://github.com/danielmichaels/go-web-app
# or with gh shorthand
uvx cookiecutter gh:danielmichaels/go-web-app
```

### Prompts

```
[1]  github_username        Your GitHub username
[2]  project_name           Human-readable project name
[3]  project_slug           Auto-derived from project_name (kebab-case)
[4]  cmd_name               CLI binary name (default: app)
[5]  project_description    Short description used in the OpenAPI spec
[6]  go_module_path         Go module path (auto-derived)
[7]  database_choice        postgres or sqlite
[8]  ci_choice              github / woodpecker / none
[9]  api_only               true drops the whole HTML layer
[10] use_tailwind           Tailwind v4 instead of the inline hand-written CSS
[11] use_pwa                manifest.json + service worker
[12] use_nats               Include NATS messaging support
[13] embed_nats             Run NATS server in-process (needs use_nats)
[14] use_river              Include River job queue
```

There is no Go version prompt: generated projects pin the current release
(1.26). GitHub Actions reads it from `go.mod`; Woodpecker and the Dockerfile
name the image tag directly.

### Quick start after generation

```shell
cd <project_slug>
task init      # templ + sqlc generate, then go mod tidy
task dev       # hot-reload dev server; starts its own Postgres
task test      # no database required
```

---

## The database model

This is the part that differs most from a conventional Go template.

| | Development / tests | Deployment |
|---|---|---|
| Where Postgres runs | Inside the process, from `./.data/postgres` | Wherever you point `DATABASE_URL` |
| How it is chosen | `EMBEDDED_POSTGRES=true` and `DATABASE_URL` empty | `DATABASE_URL` set — always wins |
| Migrations | Applied at boot by the binary | Applied at boot by the binary |
| Credentials to configure | None | The DSN |

`config.ShouldStartEmbedded` is the single decision point, and a supplied
`DATABASE_URL` always beats the embedded instance — so a deployment can never
accidentally boot a throwaway database alongside the real one. Setting
`APP_ENV=production` turns `EMBEDDED_POSTGRES=true` into a startup error.

Because migrations run in-process, the runtime image needs no `goose` binary
and no shell, so the Postgres image is `distroless/static` (~21 MB). Concurrent
replicas are safe: both the application migrations and River's own are wrapped
in Postgres advisory locks, so replicas starting together serialise instead of
colliding.

### Managing the dev database

```shell
task clean         # delete the dev database; next `task dev` migrates a fresh one
task clean:pg      # reclaim temp dirs from killed test runs
task clean:pg FORCE=1   # also stop instances that are still running
```

`task clean:pg` never kills anything by default. It reclaims what is provably
dead, reports what is alive, and tells you how to force the rest — because the
live one may be the database behind the dev server you are looking at.

> [!IMPORTANT]
> `internal/testhelpers.TruncateAll` lists application tables by hand. A new
> table that is not added there will leak rows between tests and surface as an
> unrelated flake.

---

## Options

### `database_choice`

| Value | When to use |
|---|---|
| `postgres` | The default. Embedded for dev and tests, external via `DATABASE_URL` in production. [pgx/v5](https://github.com/jackc/pgx) pool, advisory-locked migrations, shared test helper. |
| `sqlite` | Single-instance deployments, edge/VPS, no infra overhead. Uses [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (pure Go, no CGO). Litestream replication is pre-configured for S3-compatible storage, which is why this variant keeps a shell `entrypoint`. |

SQLite gets every non-database change (CI, logging, shutdown, config, UI) but
keeps Litestream and its entrypoint: Litestream has to wrap the process with
`replicate -exec`, so migrate-on-boot alone cannot replace it.

---

### `api_only`

`false` (the default) generates the HTML layer: templ templates, Datastar,
sessions and CSRF, mounted at `/app`, alongside the huma JSON API at the root.

`true` drops `internal/ui/`, `assets/static/`, and the session and templ
dependencies, leaving a pure JSON API.

CSRF uses the standard library's `http.NewCrossOriginProtection` (Go 1.25+),
which works from `Sec-Fetch-Site`/`Origin` rather than a synchronised token, so
forms need no hidden field. Rejected requests are answered **404, not 403**, so
a probe cannot use the status code to confirm a route exists. Cross-origin
callers must be listed in `TRUSTED_ORIGINS`, separated with `;` rather than
`,` — a comma is not a separator there and ends up inside the origin.

---

### `ci_choice`

| Value | What you get |
|---|---|
| `github` | `.github/workflows/ci.yml` — check → lint → test (with the Postgres download cached) → native amd64 + arm64 image builds, no QEMU → manifest merge. |
| `woodpecker` | `.woodpecker.yml` — build, lint, test, buildx to ghcr.io, plus an optional Dokploy deploy webhook. |
| `none` | No CI files. |

Both stamp the version into the binary via `-ldflags -X`, and both run
`golangci-lint` pinned to the same release, so neither can go green on code the
other rejects.

> [!NOTE]
> The Woodpecker test step runs as a non-root user on a Debian image. Neither
> is incidental: `initdb` refuses to run as root, and the embedded Postgres
> binaries are linked against glibc so they will not run on musl/alpine.

---

### `use_nats`

Adds [NATS](https://nats.io) messaging to the project — connection management, graceful shutdown, and an example queue-group subscriber.

**Use when:** you need pub/sub messaging, request-reply patterns, fan-out, or a lightweight service bus between components.

**Skip when:** the app is a straightforward REST API with no async messaging needs.

When enabled, the `internal/natsio/` package is generated with:
- `Connect` — connects to an external NATS server via `NATS_URL`
- `ExampleSubscriber` — queue-group subscriber pattern to copy from

---

### `embed_nats`

Only meaningful when `use_nats=true`. Starts a NATS server **in-process** instead of connecting to an external one.

| | `embed_nats=false` | `embed_nats=true` |
|---|---|---|
| NATS server | External (Docker, Fly, etc.) | In-process goroutine |
| Config needed | `NATS_URL` | `NATS_STORE_DIR`, `NATS_EMBED_PORT` |
| JetStream | External server config | Always enabled |
| Port | Set by external server | Random (OS-allocated) by default, override with `NATS_EMBED_PORT` |
| Best for | Production, multi-service | Self-contained single binaries, local dev without Docker |

JetStream is always enabled for embedded NATS (persistent streams, durable consumers, KV store). The store directory defaults to `data/jetstream` and is configurable via `NATS_STORE_DIR`.

---

### `use_river`

Adds [River](https://riverqueue.com) — a Go job queue backed by the same database the app already uses (no Redis, no separate broker).

**Use when:** you need background jobs, scheduled work, or retry logic and want to avoid adding another infrastructure component.

**Skip when:** you only need fire-and-forget messaging (NATS is sufficient) or the app has no background processing needs.

| Database | Driver used |
|---|---|
| `postgres` | `riverpgxv5` — uses the existing pgx pool, supports `LISTEN/NOTIFY` for instant job pickup |
| `sqlite` | `riversqlite` — poll-only mode (~1s latency), still fully functional |

River runs its own migrations at startup, under an advisory lock. Workers and
job types live in `internal/jobs/`.

Unless `api_only`, the River dashboard can be served from the same binary at
`RIVER_UI_PATH` (default `/riverui`). It is **off by default**: set
`RIVER_UI_EMBEDDED=true` to mount it.

> [!WARNING]
> The dashboard can cancel and retry jobs, and a generated project has no user
> model to authorise anyone. `Routes()` mounts it in a `chi.Group` of its own,
> inside the session and CSRF middleware, with the one line to add marked in a
> comment — put your admin check there before enabling it anywhere reachable.
> Leaving it off and running the standalone `riverui` container from
> `compose.yaml` is the other reasonable answer.

#### River vs NATS

These are complementary, not alternatives:

| | River | NATS |
|---|---|---|
| Primary use | Durable background jobs | Real-time messaging / pub-sub |
| Persistence | In your DB (survives restarts) | JetStream (optional, separate store) |
| Retry logic | Built-in with backoff | Manual |
| Scheduling | Cron support | No |
| Multi-service fan-out | No | Yes |

A common pattern is both together: NATS for real-time events, River for durable work that must complete.

---

## Security defaults

Every one of these is changeable; they are the starting position, and the
reasoning is worth knowing before overriding it.

| Default | Why |
|---|---|
| No in-process rate limiting | Behind a reverse proxy this binary sees the proxy's address, so an IP-keyed limiter here files every visitor into one bucket and lets the first busy client lock out the rest. Configure it at the edge, which knows the real caller. |
| `X_API_KEY` optional, `ApiKeyAuth` attached to nothing | A UI-only project has no endpoint worth guarding, and a mandatory secret protecting nothing only teaches operators to paste junk into config. Attach the middleware when you have something to protect; it fails closed if the key is missing by then. |
| `/metrics` on `METRICS_PORT`, never `SERVER_PORT` | It reports process statistics and enumerates every route the app answers. The Dockerfile exposes `SERVER_PORT` alone, so metrics stay reachable only from wherever the container is attached. |
| `CLIENT_IP_SOURCE=remote` | Nothing is read from forwarded headers until a deployment names the hop it trusts, because those headers are caller-controlled whenever the process is directly reachable. Behind a proxy, set `header:` or `xff:` or every access log line records the proxy. |
| `X-Frame-Options: DENY` and `frame-ancestors 'none'`, but no fuller CSP | Datastar compiles its `data-*` expressions with the `Function` constructor, which CSP counts as eval, so any `script-src` has to permit `'unsafe-eval'`. A policy that looks stricter than it is helps nobody; `securityHeaders` carries a working starting point in a comment. |
| CSRF rejects with 404, not 403 | A probe cannot use the status code to confirm that a route exists. |

## Environment variables

All config is via environment variables, decoded once by `config.Load`, which
reports **every** problem at once rather than one per restart, and refuses to
start on any of them. See `.env.example` in the generated project for the full
list — everything documented there has a reader in `internal/config`, and
nothing there describes a protection that does not exist.

---

## Development tasks

```shell
task dev                                  # hot-reload server, embedded database
task test                                 # tests, no database required
task db:migration:create -- my-migration  # new migration file
task audit                                # lint + align + format
task sqlc                                 # regenerate query code
task templ                                # regenerate templates
task clean:pg                             # reclaim embedded-Postgres leftovers
```

[uvx]: https://docs.astral.sh/uv/guides/tools/
