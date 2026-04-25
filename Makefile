SHELL := /usr/bin/env bash

.PHONY: help validate test env-check

help:
	@printf '%s\n' \
		'Developer targets:' \
		'  make validate   Run repository validation checks' \
		'  make test       Discover and run component tests when present' \
		'  make env-check  Validate environment example files'

validate: env-check test

test:
	@./scripts/run_component_tests.sh

env-check:
	@./scripts/validate_env_examples.sh
