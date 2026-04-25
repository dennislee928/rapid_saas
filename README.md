# Security Webhook Router

Repository-level developer tooling for the security webhook router monorepo.

## Quickstart

1. Review environment variables:

   ```sh
   cp .env.example .env
   ```

2. Run all repository validation checks:

   ```sh
   make validate
   ```

3. Run component tests only:

   ```sh
   make test
   ```

The validation scripts discover component manifests under `apps/`, `services/`, `db/`, `infra/`, and `packages/` when those directories exist. If no components are present yet, the test runner exits successfully with a no-op message.

## Tooling

- `Makefile` provides the stable local entrypoint.
- `scripts/run_component_tests.sh` discovers and runs tests for Node, Python, Go, Rust, and Composer components when test configuration is present.
- `scripts/validate_env_examples.sh` validates that environment examples exist and do not contain obvious committed secret values.
- `package.json` exposes equivalent npm scripts for environments that prefer a package task runner.
