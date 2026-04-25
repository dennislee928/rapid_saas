#!/usr/bin/env bash

INTERVAL_SECONDS="${INTERVAL_SECONDS:-5}"
MAX_FILE_BYTES="${MAX_FILE_BYTES:-100000000}"

echo "Starting auto-commit loop. Press Ctrl+C to stop."
echo "Interval: ${INTERVAL_SECONDS}s"
echo "Max staged file size: ${MAX_FILE_BYTES} bytes"

unstage_oversized_files() {
  local found_oversized=0

  while IFS= read -r -d '' path; do
    if [[ ! -f "$path" ]]; then
      continue
    fi

    local file_size
    file_size="$(wc -c < "$path" | tr -d '[:space:]')"

    if (( file_size > MAX_FILE_BYTES )); then
      echo "Skipping oversized file (${file_size} bytes): $path"
      git reset -q HEAD -- "$path"
      found_oversized=1
    fi
  done < <(git diff --cached --name-only -z)

  return "$found_oversized"
}

while true; do
  timestamp="$(date '+%Y-%m-%d %H:%M:%S')"
  commit_message="${COMMIT_MESSAGE:-chore: auto commit ${timestamp}}"

  echo "[$timestamp] Running git add ."
  if ! git add .; then
    echo "[$timestamp] git add failed; retrying in ${INTERVAL_SECONDS}s"
    sleep "$INTERVAL_SECONDS"
    continue
  fi

  if ! unstage_oversized_files; then
    echo "[$timestamp] Oversized staged files found; retrying in ${INTERVAL_SECONDS}s"
    sleep "$INTERVAL_SECONDS"
    continue
  fi

  echo "[$timestamp] Running git commit"
  if ! git commit -m "$commit_message"; then
    echo "[$timestamp] git commit failed; retrying in ${INTERVAL_SECONDS}s"
    sleep "$INTERVAL_SECONDS"
    continue
  fi

  echo "[$timestamp] Commit created; retrying in ${INTERVAL_SECONDS}s"
  sleep "$INTERVAL_SECONDS"
done
