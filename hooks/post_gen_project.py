import os
import shutil
import textwrap
from pathlib import Path

CWD = Path.cwd().absolute()

def create_env():
    """
    Copy .env.sample to .env automatically after creating the project.
    """

    sample_envfile: str = os.path.join(CWD, ".env.example")
    envfile: str = os.path.join(CWD, ".env")

    shutil.copyfile(sample_envfile, envfile)

def database_choice():
    """
    Create database specific files based on user choice
    """
    if "{{ cookiecutter.database_choice }}" == "postgres":
        shutil.rmtree(os.path.join(CWD, "database"))
        os.remove(os.path.join(CWD, "litestream.yml"))
    elif "{{ cookiecutter.database_choice }}" == "sqlite":
        shutil.rmtree(os.path.join(CWD, "internal/testhelpers"))
    else:
        raise ValueError("Invalid database choice")

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
    - Run the initialiser helper
        $ task init
    - Check the code works (if you have `air` in your $PATH)
        $ air
        or:
        $ task dev
        or:
        $ go run cmd/{{ cookiecutter.cmd_name.strip() }}/main.go
    - Upload initial code to git:
        $ git add -a
        $ git commit -m "Initial commit!"
        $ git remote add origin https://{{ cookiecutter.go_module_path.strip('/') }}.git
        $ git push -u origin --all
    """

    print(textwrap.dedent(message))

runners = [
    create_env,
    database_choice,
    print_final_instructions,
]

for runner in runners:
    try:
        runner()
    except ValueError as e:
        print(e)
        exit(-1)
