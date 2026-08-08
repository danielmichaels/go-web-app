#!/usr/bin/env bash
#
# Generates the template across a spread of option combinations and takes each
# one as far as a compiled binary.
#
# Most of this template's conditionals are independent, but a handful interact:
# api_only removes the whole HTML layer, so every use_tailwind and use_pwa block
# outside internal/ui has to be guarded against it too. Those pairings only
# break at generation time, which no Go test can reach.
#
#   ./scripts/check-matrix.sh            # every combination below
#   ./scripts/check-matrix.sh api-only   # one combination, named exactly
#   ./scripts/check-matrix.sh --list     # names as JSON, for the CI matrix
#
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WANTED="${1:-}"

# name | extra cookiecutter args. The defaults cover postgres + github + full UI.
COMBOS=(
  "defaults|"
  "tailwind-ui|use_tailwind=true"
  "api-only|api_only=true"
  "api-only-tailwind-pwa|api_only=true use_tailwind=true use_pwa=true"
  "sqlite-full|database_choice=sqlite use_tailwind=true use_pwa=true use_nats=true embed_nats=true ci_choice=woodpecker"
  "sqlite-api-only|database_choice=sqlite api_only=true use_river=false ci_choice=none"
  "no-river|use_river=false use_nats=true"
)

# The workflow builds its matrix from this, so the combination list lives here
# and nowhere else.
if [[ "$WANTED" == "--list" ]]; then
  sep=""
  printf '['
  for combo in "${COMBOS[@]}"; do
    printf '%s"%s"' "$sep" "${combo%%|*}"
    sep=","
  done
  printf ']\n'
  exit 0
fi

# Matching is exact, and an unknown name is an error rather than a run of
# nothing: a typo that selects no combinations would otherwise report success.
if [[ -n "$WANTED" ]]; then
  known=false
  for combo in "${COMBOS[@]}"; do
    [[ "${combo%%|*}" == "$WANTED" ]] && known=true
  done
  if [[ "$known" == false ]]; then
    echo "unknown combination: $WANTED" >&2
    echo "known: $(printf '%s ' "${COMBOS[@]%%|*}")" >&2
    exit 2
  fi
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0
failed_names=()

for combo in "${COMBOS[@]}"; do
  name="${combo%%|*}"
  args="${combo#*|}"

  if [[ -n "$WANTED" && "$name" != "$WANTED" ]]; then
    continue
  fi

  out="$WORK/$name"
  mkdir -p "$out"
  printf '%-24s ' "$name"

  log="$WORK/$name.log"
  # A subshell, not a { } group: a group runs in this shell, so the first
  # combination to fail would exit the whole run and report nothing.
  (
    # shellcheck disable=SC2086
    uvx cookiecutter "$REPO" --no-input --output-dir "$out" project_name="$name" $args || exit 1
    cd "$out/$name" || exit 1

    # `task init` itself, not a reimplementation of its steps: the failure this
    # guards against is init referencing a task, or an input file, that the
    # chosen options removed. Running the steps by hand skips exactly that.
    task init || exit 1
    go build ./... || exit 1
    test -z "$(gofmt -l . | grep -v _templ.go)" || {
      echo "gofmt reported files"
      gofmt -l . | grep -v _templ.go
      exit 1
    }
  ) >"$log" 2>&1

  if [[ $? -eq 0 ]]; then
    echo "ok"
    pass=$((pass + 1))
  else
    echo "FAIL"
    sed 's/^/    /' "$log" | tail -15
    fail=$((fail + 1))
    failed_names+=("$name")
  fi
done

echo
echo "$pass passed, $fail failed"
if ((fail > 0)); then
  echo "failed: ${failed_names[*]}"
  exit 1
fi
