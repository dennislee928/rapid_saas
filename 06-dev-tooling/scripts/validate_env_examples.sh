#!/usr/bin/env bash
set -euo pipefail

TOOL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "$TOOL_DIR/.." && pwd)"
SEARCH_DIRS=(01-ingress-worker 02-router-api 03-database 04-dashboard 05-infra-ci 06-dev-tooling)

env_examples=()
while IFS= read -r file; do
  env_examples+=("$file")
done < <(
  for dir in "${SEARCH_DIRS[@]}"; do
    [[ -d "$REPO_ROOT/$dir" ]] && find "$REPO_ROOT/$dir" -maxdepth 2 \
      -type f \( -name ".env.example" -o -name ".env.*.example" \) -print
  done | sort
)

if ((${#env_examples[@]} == 0)); then
  echo "No environment example files found."
  exit 1
fi

failed=0
for file in "${env_examples[@]}"; do
  rel="${file#"$REPO_ROOT"/}"
  echo "Checking ${rel}"

  if [[ ! -s "$file" ]]; then
    echo "  ERROR: file is empty"
    failed=1
    continue
  fi

  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" != *=* ]] && continue

    key="${line%%=*}"
    value="${line#*=}"
    key="$(printf '%s' "$key" | tr '[:lower:]' '[:upper:]')"
    value="$(printf '%s' "$value" | sed -E 's/^[[:space:]]+|[[:space:]]+$//g')"

    [[ -z "$value" ]] && continue
    [[ "$value" =~ ^(change-me|changeme|replace-me|replace_me|example|placeholder|dummy|test|localhost|127\.0\.0\.1|http://localhost|https://example\.com) ]] && continue
    [[ "$value" =~ ^(pk|sk)_test_.*(replace|example|dummy|placeholder|test) ]] && continue
    [[ "$value" =~ ^[a-z]+:// ]] && continue
    [[ "$value" =~ ^[0-9]+$ ]] && continue

    if [[ "$key" =~ (SECRET|TOKEN|KEY|PASSWORD|CREDENTIAL) ]]; then
      echo "  ERROR: ${key} appears to contain a non-placeholder value"
      failed=1
    fi
  done < "$file"
done

exit "$failed"
