#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SEARCH_DIRS=(apps services db infra packages)

declare -a commands=()
seen_keys=$'\n'

add_command() {
  local dir="$1"
  local cmd="$2"
  commands+=("$dir"$'\t'"$cmd")
}

mark_seen() {
  local key="$1"
  case "$seen_keys" in
    *$'\n'"$key"$'\n'*) return 0 ;;
    *)
      seen_keys="${seen_keys}${key}"$'\n'
      return 1
      ;;
  esac
}

has_package_script() {
  local manifest="$1"
  local script_name="$2"

  if command -v node >/dev/null 2>&1; then
    node -e '
      const fs = require("fs");
      const manifest = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
      process.exit(manifest.scripts && manifest.scripts[process.argv[2]] ? 0 : 1);
    ' "$manifest" "$script_name"
    return
  fi

  grep -Eq "\"${script_name}\"[[:space:]]*:" "$manifest"
}

node_test_command() {
  local dir="$1"

  if [[ -f "$ROOT_DIR/pnpm-lock.yaml" || -f "$dir/pnpm-lock.yaml" ]]; then
    printf 'pnpm test'
  elif [[ -f "$ROOT_DIR/yarn.lock" || -f "$dir/yarn.lock" ]]; then
    printf 'yarn test'
  else
    printf 'npm test'
  fi
}

discover_node() {
  local manifest dir
  while IFS= read -r manifest; do
    dir="$(dirname "$manifest")"
    [[ "$dir" == "$ROOT_DIR" ]] && continue
    mark_seen "node:$dir" && continue

    if has_package_script "$manifest" test; then
      add_command "$dir" "$(node_test_command "$dir")"
    fi
  done < <(find_existing -name package.json)
}

discover_python() {
  local manifest dir
  while IFS= read -r manifest; do
    dir="$(dirname "$manifest")"
    mark_seen "python:$dir" && continue

    if [[ -d "$dir/tests" || -f "$dir/pytest.ini" || -f "$dir/setup.cfg" ]]; then
      add_command "$dir" "python3 -m pytest"
    fi
  done < <(find_existing \( -name pyproject.toml -o -name pytest.ini -o -name setup.cfg \))
}

discover_go() {
  local manifest dir
  while IFS= read -r manifest; do
    dir="$(dirname "$manifest")"
    add_command "$dir" "go test ./..."
  done < <(find_existing -name go.mod)
}

discover_rust() {
  local manifest dir
  while IFS= read -r manifest; do
    dir="$(dirname "$manifest")"
    add_command "$dir" "cargo test"
  done < <(find_existing -name Cargo.toml)
}

discover_composer() {
  local manifest dir
  while IFS= read -r manifest; do
    dir="$(dirname "$manifest")"
    if grep -Eq '"test"[[:space:]]*:' "$manifest"; then
      add_command "$dir" "composer test"
    fi
  done < <(find_existing -name composer.json)
}

find_existing() {
  local existing=()
  local dir

  for dir in "${SEARCH_DIRS[@]}"; do
    [[ -d "$ROOT_DIR/$dir" ]] && existing+=("$ROOT_DIR/$dir")
  done

  ((${#existing[@]} == 0)) && return 0
  find "${existing[@]}" \
    \( -name node_modules -o -name .next -o -name .turbo -o -name dist -o -name build -o -name target -o -name vendor \) -prune -o \
    -type f "$@" -print | sort
}

run_commands() {
  local entry dir cmd rel

  if ((${#commands[@]} == 0)); then
    echo "No component tests discovered."
    return 0
  fi

  for entry in "${commands[@]}"; do
    dir="${entry%%$'\t'*}"
    cmd="${entry#*$'\t'}"
    rel="${dir#"$ROOT_DIR"/}"

    echo "==> ${rel}: ${cmd}"
    (cd "$dir" && eval "$cmd")
  done
}

discover_node
discover_python
discover_go
discover_rust
discover_composer
run_commands
