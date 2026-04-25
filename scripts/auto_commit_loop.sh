#!/usr/bin/env bash

INTERVAL_SECONDS="${INTERVAL_SECONDS:-5}"

echo "Starting auto-commit loop. Press Ctrl+C to stop."
echo "Interval: ${INTERVAL_SECONDS}s"

while true; do
  timestamp="$(date '+%Y-%m-%d %H:%M:%S')"
  commit_message="${COMMIT_MESSAGE:-chore: auto commit ${timestamp}}"

  echo "[$timestamp] Running git add ."
  if ! git add .; then
    echo "[$timestamp] git add failed; retrying in ${INTERVAL_SECONDS}s"
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
