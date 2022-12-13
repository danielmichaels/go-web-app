module {{ cookiecutter.go_module_path.strip('/') }}

go {{ cookiecutter.go_version }}

require (
github.com/go-chi/chi/v5 v5.0.7
github.com/go-chi/httplog v0.2.5
github.com/go-mail/mail/v2 v2.3.0
github.com/joeshaw/envdecode v0.0.0-20200121155833-099f1fc765bd
github.com/rs/zerolog v1.28.0
golang.org/x/exp v0.0.0-20221114191408-850992195362
golang.org/x/text v0.4.0
)

require (
github.com/mattn/go-colorable v0.1.12 // indirect
github.com/mattn/go-isatty v0.0.16 // indirect
golang.org/x/sys v0.2.0 // indirect
gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
gopkg.in/mail.v2 v2.3.1 // indirect
)
