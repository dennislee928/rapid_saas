SHELL := /usr/bin/env bash
TOOL_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

.PHONY: help validate test env-check

help:
	@printf '%s\n' \
		'Developer targets:' \
		'  make validate   Run repository validation checks' \
		'  make test       Discover and run component tests when present' \
		'  make env-check  Validate environment example files'

validate: env-check test

test:
	@$(TOOL_DIR)/scripts/run_component_tests.sh

env-check:
	@$(TOOL_DIR)/scripts/validate_env_examples.sh
