# go-web-app

> A Go web application boilerplate with options to add NATS, API servers and databases

## Server setup

To run the server:

```shell
air
# OR
# task dev
```

## Requirements

This expects at least the following:

- [goose](https://github.com/pressly/goose)
- [sqlc](https://sqlc.dev)
- [air](https://github.com/cosmtrek/air)
- [task](https://taskfile.dev)

The rest will be installed during `go mod tidy`.

## Assets setup

The CSS and JS requires some manual building occasionally.

A `Makefile` helper exists to do both of the following in a single command.
`task assets` will regenerate new bundles.
