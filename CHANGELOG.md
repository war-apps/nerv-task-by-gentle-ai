# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
