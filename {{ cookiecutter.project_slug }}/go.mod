module {{ cookiecutter.go_module_path.strip('/') }}

go {{ cookiecutter.go_version }}

require (
	github.com/a-h/templ v0.2.747
	github.com/alecthomas/kong v0.9.0
	github.com/alecthomas/kong-yaml v0.2.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-chi/httplog v0.3.2
	github.com/go-chi/render v1.0.3
	github.com/joeshaw/envdecode v0.0.0-20200121155833-099f1fc765bd
	github.com/rs/zerolog v1.28.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/tools v0.10.0 // indirect
	lukechampine.com/uint128 v1.3.0 // indirect
	modernc.org/cc/v3 v3.41.0 // indirect
	modernc.org/ccgo/v3 v3.16.14 // indirect
	modernc.org/libc v1.24.1 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.6.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.1.0 // indirect
)

require (
	github.com/go-playground/form/v4 v4.2.0
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/pressly/goose/v3 v3.13.0
	github.com/spf13/cobra v1.7.0
	golang.org/x/sys v0.9.0 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/mail.v2 v2.3.1 // indirect
	modernc.org/sqlite v1.23.1
)
