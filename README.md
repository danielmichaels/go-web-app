# go-web-app

> A [cookiecutter] for quickly scaffolding go web applications

This is a **highly** opinionated way of creating a Go web app. After manually
creating these time and time again this is how I've settled on bootstrapping
new app's. 

The application uses Go templates but can also be just an API backend; `go-chi`
is the router and is highly extensible.

## Usage

To create a new go-chi web app using this repository you only need to run the following:

```shell
cookiecutter https://github.com/danielmichaels/go-web-app
# or gh:danielmichaels/go-web-app
```
And then answer the prompts. Here's an example run:

```shell
z ❯ cookiecutter https://github.com/danielmichaels/go-web-app
github_username [danielmichaels]: 
project_name [go-web-app]: demo
cmd_name [app]: api
project_description [A short description]: 
go_module_path [github.com/danielmichaels/demo]: 
Select go_version:
1 - 1.18
2 - 1.19
Choose from 1, 2 [1]: 2
```

This will create a directory called `demo` in the current working directory. All upper case
letters are converted to lowercase and hypens are used instead of spaces.

After `cookiecutter` has run the following output will be printed to the screen detailing
what to do next.

```shell
====================================================================================
Your project `demo` is ready!
The following is a *brief* overview of steps to push code to remote and
how to get your go module working.
- Move to project directory, and initialize a git repository:
    $ cd demo && git init
- Run go mod tidy to pull in dependencies:
    $ go mod tidy
- If you want to update upstream dependencies (optional; recommended)
    $ go get -u
- Check the code works
    $ go run cmd/app/main.go
- Upload initial code to git:
    $ git add -a
    $ git commit -m "Initial commit!"
    $ git remote add origin https://github.com/danielmichaels/demo.git
    $ git push -u origin --all
```
