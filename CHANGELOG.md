# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.1] - 2026-11-16

### Fixed

- `nerv-task install` now refreshes every nerv-task plugin that is
  already enabled in settings.json (`<name>@nerv-task`), not just
  task@nerv-task and task-teamwork@nerv-task when `--with-teamwork` is
  passed. `nerv-task upgrade` re-runs install with no flags and no
  terminal, so an already-enabled plugin like task-teamwork@nerv-task
  was silently left at its old installed version after an upgrade; it
  is now uninstalled and reinstalled like the core plugin. Never enables
  a plugin the caller did not already have.

## [1.1.0] - 2026-11-16

### Added

- `nerv-task version [--json]` prints the binary and task plugin versions, following the NERV ecosystem version contract (`nerv version`'s own format/fields) so `nerv` can report nerv-task as installed/outdated.

## [1.0.1] - 2026-11-16

### Fixed

- `nerv-task install` no longer aborts a first install when uninstalling
  task@nerv-task fails because it was never installed yet
  (`Plugin "..." not found in installed plugins`); installed_plugins.json
  is checked first instead of depending on the CLI's wording.
- `nerv-task install` now installs task-teamwork@nerv-task automatically
  when task-teamwork@nerv was enabled before install, even without
  `--with-teamwork` and without a terminal to offer it, so the Teamwork
  provider is never silently lost. The old nerv plugins are disabled, and
  settings.json written, only after the new plugins installed
  successfully, so a failed install leaves task@nerv/task-teamwork@nerv
  enabled.
- Bumped `nerv-kit-by-gentle-ai` to v0.1.2: the materialized
  marketplace.json now lists the core plugin as `task` (its own
  plugin.json name), not `nerv-task` (the marketplace name).

## [1.0.0] - 2026-11-16

First release: nerv's task extension as an independent tool of the NERV
ecosystem. It installs with `scripts/install.sh` or `scripts/install.ps1` and
works without nerv; on a machine with nerv's own `task@nerv` plugin,
`nerv-task install` disables it so `/task:*` is not duplicated.

### Added

- `nerv-task install`, `nerv-task configure`, `nerv-task init` and
  `nerv-task upgrade` commands.
- The `nerv-task` plugin marketplace: the `task` plugin (`/task:*`
  commands, the `task:dispatcher` agent, the `task` skill contract) and
  the `task-teamwork` provider.
- The provider manifest contract (`internal/provider`): parsing,
  validation and MCP tool-availability resolution for `provider.yaml`.
