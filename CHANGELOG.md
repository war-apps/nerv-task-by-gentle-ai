# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
