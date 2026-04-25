# Developer Tooling

Developer tooling for the security webhook router six-folder repository layout.

## Quickstart

Run these commands from this folder:

```sh
cd 06-dev-tooling
make validate
```

Or run the same checks through npm without installing dependencies here:

```sh
npm run validate
```

## Tooling

- `Makefile` provides the stable local entrypoint for developer checks.
- `scripts/run_component_tests.sh` discovers component tests in `01-ingress-worker/`, `02-router-api/`, `03-database/`, `04-dashboard/`, and `05-infra-ci/`.
- `scripts/validate_env_examples.sh` checks environment example files in the six numbered folders for obvious committed secrets.
- `package.json` exposes equivalent npm scripts for task-runner users.

## Layout Assumptions

The repository root is the parent directory of `06-dev-tooling/`.

The component test runner intentionally avoids root-level package installation and root `node_modules` writes. Node components are tested only when they define a `test` script and have a component-local `node_modules/.bin` directory available. Go build and module caches are redirected into `06-dev-tooling/.cache/`.

If a component has no test configuration or lacks local dependencies, the runner reports that condition and continues unless a discovered test command fails.
