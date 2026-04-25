#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

env_examples=()
while IFS= read -r file; do
  env_examples+=("$file")
done < <(
  find "$ROOT_DIR" -maxdepth 1 \
    -type f \( -name ".env.example" -o -name ".env.*.example" \) -print | sort
)

if ((${#env_examples[@]} == 0)); then
  echo "No environment example files found."
  exit 1
fi

failed=0
for file in "${env_examples[@]}"; do
  rel="${file#"$ROOT_DIR"/}"
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
    [[ "$value" =~ ^(change-me|changeme|example|placeholder|dummy|test|localhost|127\.0\.0\.1|http://localhost|https://example\.com) ]] && continue
    [[ "$value" =~ ^[a-z]+:// ]] && continue
    [[ "$value" =~ ^[0-9]+$ ]] && continue

    if [[ "$key" =~ (SECRET|TOKEN|KEY|PASSWORD|CREDENTIAL) ]]; then
      echo "  ERROR: ${key} appears to contain a non-placeholder value"
      failed=1
    fi
  done < "$file"
done

exit "$failed"
