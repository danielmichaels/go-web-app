import os
import shutil
import textwrap
from pathlib import Path

CWD = Path.cwd().absolute()

def create_env():
    """
    Copy .env.sample to .env automatically after creating the project.
    """

    sample_envfile: str = os.path.join(CWD, ".env.sample")
    envfile: str = os.path.join(CWD, ".env")

    shutil.copyfile(sample_envfile, envfile)

def print_final_instructions():
    """
    Simply prints final instructions for users to follow once they generate a project
    using this template!
    """
    message = """
    ====================================================================================
    Your project `{{ cookiecutter.project_name.strip() }}` is ready!
    The following is a *brief* overview of steps to push code to remote and
    how to get your go module working.
    - Move to project directory, and initialize a git repository:
        $ cd {{ cookiecutter.project_name.strip() }} && git init
    - Run go mod tidy to pull in dependencies:
        $ go mod tidy
    - Create node resources (Tailwind and Alpine.js)
        $ yarn
        $ make assets
    - Check the code works (if you have `air` in your $PATH)
        $ air
        or:
        $ go run cmd/{{ cookiecutter.cmd_name.strip() }}/main.go
        or:
        $ make run/{{ cookiecutter.cmd_name.strip() }}
    - Upload initial code to git:
        $ git add -a
        $ git commit -m "Initial commit!"
        $ git remote add origin https://{{ cookiecutter.go_module_path.strip('/') }}.git
        $ git push -u origin --all
    """

    print(textwrap.dedent(message))

runners = [
    create_env,
    print_final_instructions,
]

for runner in runners:
    try:
        runner()
    except ValueError as e:
        print(e)
        exit(-10)
