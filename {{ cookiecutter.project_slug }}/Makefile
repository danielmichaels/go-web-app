PROJECT_NAME={{ cookiecutter.project_name.strip() }}
include .env
current_time = $(shell date --iso-8601=seconds)
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.buildTime=${current_time} -X main.version=${git_description}'

.PHONY: help

default: help

.PHONY: confirm
confirm:
	@echo -n "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]

## help: Print commands help.
.PHONY: help
help : Makefile
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## env: print environment variables (makefile sanity check)
.PHONY: env
env:
	env

## run/{{ cookiecutter.cmd_name.strip() }}: Run the server
.PHONY: run/{{ cookiecutter.cmd_name.strip() }}
run/{{ cookiecutter.cmd_name.strip() }}:
	@go run ./cmd/{{ cookiecutter.cmd_name.strip() }}/main.go

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/development: Run the server with hot-reloading (uses `air`)
.PHONY: run/development
run/development:
	@air

## run/docgen: Create new routes.md
.PHONY: run/docgen
run/docgen:
	@go run ./cmd/{{ cookiecutter.cmd_name.strip() }} -docgen > _docs/routes.md
# ==================================================================================== #
# MIGRATIONS
# ==================================================================================== #

## db/migrations/up: Run migration up (apply migrations)
.PHONY: db/migration/up
db/migration/up: confirm
	@echo "setting up the database extensions..."
	goose -dir ./db/migrations sqlite3 ${DATABASE_URL} up

## db/migration/down/to version=$1: Roll back migration to $1
.PHONY: db/migration/down/to
db/migration/down/to: confirm
	@echo "rolling back migrations to ${version}"
	goose -dir ./db/migrations sqlite3 ${DATABASE_URL} down-to ${version}

## db/migration/down: drop all migrations
.PHONY: db/migration/down
db/migration/down: confirm
	@echo "rolling back all migrations"
	goose -dir ./migrations sqlite3 ${DATABASE_URL} down

## db/migration/redo: rollback latest migration, then reapply
.PHONY: db/migration/redo
db/migration/redo: confirm
	@echo "rolling back latest migration, then re-applying it; redo"
	goose -dir ./db/migrations sqlite3 ${DATABASE_URL} redo

## db/migration/create name=$1: Create new migrations with name of $1
.PHONY: db/migration/create
db/migration/create: confirm
	@echo "creating migration files for ${name}"
	goose -dir ./db/migrations sqlite3 $(DATABASE_URL) create ${name} sql
	goose -dir ./db/migrations sqlite3 $(DATABASE_URL) fix

## db/migration/status: Check database migration status
.PHONY: db/migration/status
db/migration/status:
	@echo "checking migration status for ${DATABASE_URL}"
	goose -dir ./db/migrations sqlite3 ${DATABASE_URL} status

## db/migration/reset: Roll back all migrations
.PHONY: db/migration/reset
db/migration/reset: confirm
	@echo "rolling back all migrations on ${DATABASE_URL}"
	goose -dir ./db/migrations sqlite3 ${DATABASE_URL} reset

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Vetting code..."
	go vet ./...
	staticcheck ./...
	@echo "Running tests..."
	CGO_ENABLED=1 go test -race -vet=off ./...

## test: run tests with coverage
.PHONY: test
test:
	@echo "Running tests..."
	CGO_ENABLED=1 go test -race -cover -vet=off ./...

## tparse: run tests with coverage using tparse
.PHONY: tparse
tparse:
	@CGO_ENABLED=1 go test -race -cover -vet=off ./... -json | tparse -notests
## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo "Tidying and verifying module dependencies..."
	go mod tidy
	go mod verify
	@echo "Vendoring dependencies..."
	go mod vendor

## assets: build assets
.PHONY: assets
assets:
	@echo "Building CSS and JS asset bundles..."
	yarn tailwind
	yarn alpine
