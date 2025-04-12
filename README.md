# go-web-app

> A [cookiecutter](https://cookiecutter.readthedocs.io/en/stable/) for quickly scaffolding go web applications

This is a **highly** opinionated way of creating a Go web app. After manually
creating these time and time again this is how I've settled on bootstrapping
new app's. 

This application uses:

- [huma](https://huma.rocks)
- [NATS](https://nats.io)

With supporting tools:
- [goose](https://github.com/pressly/goose)
- [sqlc](https://sqlc.dev)
- [air](https://github.com/cosmtrek/air)
- [task](https://taskfile.dev)

NATS is optional. Sqlite is the default database but Postgres is an option.

See options below for more details.

## Usage

> [!NOTE]
> I recommend [uxv] to call cookiecutter


To create a new web app using this repository you only need to run the following:

```shell
cookiecutter https://github.com/danielmichaels/go-web-app
# or gh:danielmichaels/go-web-app
```
And then answer the prompts. Here's an example run using the defaults:

```shell
# I recommend uxv to call cookiecutter
z ❯ uvx cookiecutter https://github.com/danielmichaels/go-web-app
  [1/9] github_username (danielmichaels): 
  [2/9] project_name (go-web-app): 
  [3/9] project_slug (go-web-app): 
  [4/9] cmd_name (app): 
  [5/9] project_description (A Go web application boilerplate with options to add NATS, API servers and databases): 
  [6/9] go_module_path (github.com/danielmichaels/go-web-app): 
  [7/9] use_nats [y/n] (y): 
  [8/9] Select database_choice
    1 - sqlite
    2 - postgres
    Choose from [1/2] (1): 
  [9/9] Select go_version
    1 - 1.24
    2 - 1.23
    Choose from [1/2] (1): 

```

This will create a directory called `go-web-app` in the current working directory. All upper case
letters are converted to lowercase and hypens are used instead of spaces.

After `cookiecutter` has run the following output will be printed to the screen detailing
what to do next.

```shell
====================================================================================
Your project `go-web-app` is ready!
The following is a *brief* overview of steps to push code to remote and
how to get your go module working.
- Move to project directory, and initialize a git repository:
    $ cd go-web-app && git init
- Run go mod tidy to pull in dependencies:
    $ go mod tidy
- If you want to update upstream dependencies (optional; recommended)
    $ go get -u
- Create node resources (Tailwind and Alpine.js)
    $ yarn
    $ make assets
- Check the code works (if you have `air` in your $PATH)
    $ air
    or:
    $ go run cmd/app/main.go
    or:
    $ make run/app
- Upload initial code to git:
    $ git add -a
    $ git commit -m "Initial commit!"
    $ git remote add origin https://github.com/danielmichaels/go-web-app.git
    $ git push -u origin --all
```

[uvx]: https://docs.astral.sh/uv/guides/tools/