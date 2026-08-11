import os
import shutil
import subprocess
import textwrap
import zlib
from pathlib import Path

CWD = Path.cwd().absolute()

PROJECT_SLUG = "{{ cookiecutter.project_slug }}"
DB_CHOICE = "{{ cookiecutter.database_choice }}"
CI_CHOICE = "{{ cookiecutter.ci_choice }}"
API_ONLY = "{{ cookiecutter.api_only }}" == "True"
USE_TAILWIND = "{{ cookiecutter.use_tailwind }}" == "True"
USE_PWA = "{{ cookiecutter.use_pwa }}" == "True"
USE_NATS = "{{ cookiecutter.use_nats }}" == "True"
EMBED_NATS = "{{ cookiecutter.embed_nats }}" == "True"
USE_RIVER = "{{ cookiecutter.use_river }}" == "True"


def remove(*relpaths):
    """Delete files or directories, ignoring ones that are already absent."""
    for rel in relpaths:
        target = CWD / rel
        if target.is_dir():
            shutil.rmtree(target)
        elif target.exists():
            target.unlink()


# The band sits below every OS ephemeral range -- Linux allocates from 32768,
# macOS from 49152 -- so a port reserved here is never one the kernel might
# hand to something else first.
EMBEDDED_PORT_BASE = 15432
EMBEDDED_PORT_SPAN = 1000


def assign_embedded_port():
    """Give this project its own Postgres port, derived from its slug.

    Every project sharing one default port meant two of them could not run at
    once, and the second silently attached to the first's database instead of
    starting its own.
    """
    if DB_CHOICE != "postgres":
        return

    port = EMBEDDED_PORT_BASE + zlib.crc32(PROJECT_SLUG.encode()) % EMBEDDED_PORT_SPAN
    edits = {
        ".env.example": ("EMBEDDED_POSTGRES_PORT=5433", f"EMBEDDED_POSTGRES_PORT={port}"),
        "internal/config/config.go": (
            "EMBEDDED_POSTGRES_PORT,default=5433",
            f"EMBEDDED_POSTGRES_PORT,default={port}",
        ),
        "Taskfile.yaml": ("localhost:5433", f"localhost:{port}"),
    }

    for rel, (old, new) in edits.items():
        path = CWD / rel
        text = path.read_text()
        if old not in text:
            raise ValueError(f"post_gen: expected {old!r} in {rel}; port not applied")
        path.write_text(text.replace(old, new))


def create_env():
    shutil.copyfile(CWD / ".env.example", CWD / ".env")


def database_choice():
    """
    Keep only the files belonging to the chosen database.

    Postgres runs embedded for dev and tests, so it needs internal/embeddedpg
    and the orphan-sweep script but no Litestream. SQLite replicates with
    Litestream and keeps the shell entrypoint that wraps the process.
    """
    if DB_CHOICE == "postgres":
        remove("database", "litestream.yml", "entrypoint", "internal/store/sqlite_test.go")
    elif DB_CHOICE == "sqlite":
        remove(
            "internal/embeddedpg",
            "internal/testhelpers",
            "internal/store/advisory_lock.go",
            "scripts/clean-pg.sh",
            # Both drive the shared embedded-Postgres helper.
            "internal/store/store_test.go",
            "internal/server/routes_test.go",
        )
    else:
        raise ValueError(f"Invalid database choice: {DB_CHOICE}")

    handle_dockerfiles()


def handle_dockerfiles():
    """Promote the chosen Dockerfile to the project root and drop the rest."""
    docker_dir = CWD / "docker"
    chosen = docker_dir / f"Dockerfile.{DB_CHOICE}"
    if chosen.exists():
        shutil.move(str(chosen), str(CWD / "Dockerfile"))
    remove("docker")


def handle_compose():
    """
    Drop compose.yaml when it would declare no services.

    SQLite is a local file and the job dashboard only ships for Postgres, so a
    SQLite project without NATS has nothing to compose.
    """
    if DB_CHOICE == "sqlite" and not USE_NATS:
        remove("compose.yaml")


def handle_ci_choice():
    if CI_CHOICE == "github":
        remove(".woodpecker.yml")
    elif CI_CHOICE == "woodpecker":
        remove(".github")
    elif CI_CHOICE == "none":
        remove(".github", ".woodpecker.yml")
    else:
        raise ValueError(f"Invalid CI choice: {CI_CHOICE}")


def handle_web_ui():
    """
    Strip the rendering layer for an API-only service.

    assets/static goes with it: assets/embed.go only embeds that directory
    when a UI is present, so leaving an orphaned one behind would be dead
    weight in the binary.
    """
    if API_ONLY:
        remove("internal/ui", "assets/static", "assets/css")
        # Say so rather than leaving the answer to look accepted: both options
        # only ever apply to the HTML layer that api_only just removed.
        ignored = [
            name
            for name, chosen in (("use_tailwind", USE_TAILWIND), ("use_pwa", USE_PWA))
            if chosen
        ]
        if ignored:
            print(
                f"note: api_only drops the HTML layer, so {' and '.join(ignored)} "
                "had nothing to apply to and was ignored."
            )
        return

    # Neither build serves a stylesheet from the tree as committed: Tailwind
    # generates this file with `task css`, and the hand-written stylesheet is
    # inline in layout.templ so a fresh project cannot render unstyled.
    remove("assets/static/css/main.css")
    if not USE_TAILWIND:
        remove("assets/css")
    if not USE_PWA:
        remove("assets/static/manifest.json", "assets/static/sw.js")


def handle_nats_package():
    if not USE_NATS:
        remove("internal/natsio")
    elif not EMBED_NATS:
        remove("internal/natsio/embed.go")


def handle_river_package():
    if not USE_RIVER:
        remove("internal/jobs")


def prune_empty_dirs():
    """Git cannot track an empty directory, so do not leave one behind."""
    for rel in ("scripts", "assets/static/js", "assets/static/css", "assets/static"):
        target = CWD / rel
        if target.is_dir() and not any(target.iterdir()):
            target.rmdir()


def format_go_sources():
    """Format the rendered Go sources.

    Struct field alignment and import order both depend on which conditional
    blocks fired, so no template can be written that is correct for every
    combination of options. Formatting after rendering is the only way to get
    them right, and without it a fresh project fails its own `task audit`.
    """
    gofmt = shutil.which("gofmt")
    if gofmt is None:
        print("post_gen: gofmt not on PATH; generated sources left unformatted")
        return

    result = subprocess.run([gofmt, "-w", str(CWD)], capture_output=True, text=True)
    if result.returncode != 0:
        print(f"post_gen: gofmt failed, sources left unformatted:\n{result.stderr}")


def print_final_instructions():
    message = """
    ====================================================================================
    Your project `{{ cookiecutter.project_name.strip() }}` is ready!

    - Move to the project directory and initialise a git repository:
        $ cd {{ cookiecutter.project_slug }} && git init
    - Install the tools and generate code:
        $ task init
    - Run it. No database to install: {% if cookiecutter.database_choice == 'postgres' %}Postgres runs embedded from ./.data/postgres{% else %}SQLite lives in ./database{% endif %}
        $ task dev
    - Run the tests. Also no database required:
        $ task test
    - Upload the initial code:
        $ git add -A
        $ git commit -m "Initial commit!"
        $ git remote add origin https://{{ cookiecutter.go_module_path.strip('/') }}.git
        $ git push -u origin --all
    """

    print(textwrap.dedent(message))


runners = [
    assign_embedded_port,
    create_env,
    database_choice,
    handle_compose,
    handle_ci_choice,
    handle_web_ui,
    handle_nats_package,
    handle_river_package,
    prune_empty_dirs,
    format_go_sources,
    print_final_instructions,
]

for runner in runners:
    try:
        runner()
    except ValueError as e:
        print(e)
        exit(-1)
