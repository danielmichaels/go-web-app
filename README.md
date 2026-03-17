# go-web-app

> A [cookiecutter](https://cookiecutter.readthedocs.io/en/stable/) for quickly scaffolding Go web applications

An opinionated Go web app template. Batteries included, production-ready defaults, optional components selected at generation time.

## Stack

| Component | Library |
|---|---|
| HTTP framework | [chi](https://github.com/go-chi/chi) |
| API / OpenAPI | [huma](https://huma.rocks) |
| CLI | [kong](https://github.com/alecthomas/kong) |
| Logging | [slog](https://pkg.go.dev/log/slog) + [slogcolor](https://github.com/SladkyCitron/slogcolor) |
| Migrations | [goose](https://github.com/pressly/goose) |
| Query gen | [sqlc](https://sqlc.dev) |
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
[6]  go_module_path         Go module path (auto-derived from github_username + project_name)
[7]  use_nats               Include NATS messaging support
[8]  embed_nats             Run NATS server in-process (only relevant if use_nats=true)
[9]  database_choice        sqlite or postgres
[10] go_version             1.26 / 1.25 / 1.24
[11] use_river              Include River job queue
```

### Quick start after generation

```shell
cd <project_slug>
task init      # go mod tidy + sqlc generate
task serve     # hot-reload dev server via air
```

---

## Options

### `database_choice`

| Value | When to use |
|---|---|
| `sqlite` | Single-instance deployments, edge/VPS, no infra overhead. Uses [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (pure Go, no CGO). Litestream replication is pre-configured for S3-compatible storage. |
| `postgres` | Multi-instance, high-write workloads, or when you need advanced SQL features. Includes [pgx/v5](https://github.com/jackc/pgx) connection pool and [testcontainers](https://golang.testcontainers.org) integration test helpers. |

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

**Use `embed_nats=true` when:** you want a fully self-contained binary with no external dependencies — common for SQLite-backed edge deployments where the entire app ships as a single binary.

**Use `embed_nats=false` when:** multiple services share the same NATS server, or you need NATS clustering/HA.

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

River runs database migrations automatically on startup via `rivermigrate`. Workers and job types are defined in `internal/jobs/`.

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

## Environment variables

All config is via environment variables. See `.env.example` in the generated project for the full list with defaults, grouped by section (server, logging, database, NATS, River).

---

## Development tasks

```shell
task serve                                 # hot-reload server
task db:migration:create -- my-migration  # new migration file
task db:migration:up                      # run migrations
task audit                                # lint + align + format
task sqlc                                 # regenerate query code
```

[uvx]: https://docs.astral.sh/uv/guides/tools/
