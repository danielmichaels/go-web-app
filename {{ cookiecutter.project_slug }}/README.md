# {{ cookiecutter.project_name.strip() }}

> {{ cookiecutter.project_description.strip() }}

## Server setup

To run the server:

```shell
air
# OR
# make run/development
```

## Requirements

This expects at least the following:

- [goose](https://github.com/pressly/goose)
- [sqlc](https://sqlc.dev)
- [air](https://github.com/cosmtrek/air)

The rest will be installed during `go mod tidy`.

## Assets setup

The CSS and JS requires some manual building occasionally.

A `Makefile` helper exists to do both of the following in a single command.
`make assets` will regenerate new bundles.
