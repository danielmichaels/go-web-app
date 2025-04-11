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
    db_choice = "{{ cookiecutter.database_choice }}"
    if db_choice == "postgres":
        shutil.rmtree(os.path.join(CWD, "database"))
        os.remove(os.path.join(CWD, "litestream.yml"))
    elif db_choice == "sqlite":
        shutil.rmtree(os.path.join(CWD, "internal/testhelpers"))
    else:
        raise ValueError("Invalid database choice")
    handle_dockerfiles(db_choice)

def handle_dockerfiles(db_choice):
    """
    Handle Docker files based on database choice.
    Keeps the appropriate Dockerfile and removes the other.
    """
    docker_dir = os.path.join(CWD, "zarf/docker")
    sqlite_dockerfile = os.path.join(docker_dir, "Dockerfile.sqlite")
    postgres_dockerfile = os.path.join(docker_dir, "Dockerfile.postgres")
    target_dockerfile = os.path.join(docker_dir, "Dockerfile")

    if db_choice == "sqlite":
        if os.path.exists(postgres_dockerfile):
            os.remove(postgres_dockerfile)
        if os.path.exists(sqlite_dockerfile):
            shutil.move(sqlite_dockerfile, target_dockerfile)
    elif db_choice == "postgres":
        if os.path.exists(sqlite_dockerfile):
            os.remove(sqlite_dockerfile)
        if os.path.exists(postgres_dockerfile):
            shutil.move(postgres_dockerfile, target_dockerfile)
    else:
        raise ValueError("Invalid database choice")


def handle_compose_directory():
    """
    Delete the zarf/compose directory if use_nats is false and database_choice is sqlite.
    This is because the compose setup is not needed in this configuration.
    """
    use_nats = "{{ cookiecutter.use_nats }}"
    db_choice = "{{ cookiecutter.database_choice }}"

    if not use_nats and db_choice == "sqlite":
        compose_dir = os.path.join(CWD, "zarf/compose")
        if os.path.exists(compose_dir):
            shutil.rmtree(compose_dir)
            print("Removed zarf/compose directory as it's not needed with SQLite and no NATS")

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
    handle_compose_directory,
    print_final_instructions,
]

for runner in runners:
    try:
        runner()
    except ValueError as e:
        print(e)
        exit(-1)
