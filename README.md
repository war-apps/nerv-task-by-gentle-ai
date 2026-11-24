# nerv-task

<p align="center">
  <a href="https://code.claude.com/docs"><img alt="Claude Code plugin" src="https://img.shields.io/badge/Claude_Code-plugin-d97757?style=for-the-badge&logo=anthropic&logoColor=white"></a>
  <a href="https://go.dev/"><img alt="Go CLI" src="https://img.shields.io/badge/Go-CLI-00ADD8?style=for-the-badge&logo=go&logoColor=white"></a>
  <a href="https://www.markdownguide.org/"><img alt="Markdown commands and agent" src="https://img.shields.io/badge/Markdown-commands_%26_agent-000000?style=for-the-badge&logo=markdown&logoColor=white"></a>
  <a href="https://yaml.org/"><img alt="YAML config" src="https://img.shields.io/badge/YAML-task.yaml-CB171E?style=for-the-badge&logo=yaml&logoColor=white"></a>
  <a href="https://www.teamwork.com/"><img alt="Teamwork provider" src="https://img.shields.io/badge/Teamwork-MCP_provider-6c5ce7?style=for-the-badge"></a>
  <a href="https://github.com/war-apps/nerv-by-gentle-ai"><img alt="NERV ecosystem" src="https://img.shields.io/badge/NERV-ecosystem_tool-8a2be2?style=for-the-badge"></a>
  <a href="https://github.com/war-apps/nerv-kit-by-gentle-ai"><img alt="nerv-kit" src="https://img.shields.io/badge/nerv--kit-shared_library-5a4fcf?style=for-the-badge"></a>
  <a href="https://www.gnu.org/software/bash/"><img alt="Bash and PowerShell installers" src="https://img.shields.io/badge/install-sh_%7C_ps1-4EAA25?style=for-the-badge&logo=gnubash&logoColor=white"></a>
  <a href="https://goreleaser.com/"><img alt="GoReleaser" src="https://img.shields.io/badge/GoReleaser-releases-0F9D9A?style=for-the-badge"></a>
  <a href="https://git-scm.com/"><img alt="git" src="https://img.shields.io/badge/git-gitflow-F05032?style=for-the-badge&logo=git&logoColor=white"></a>
</p>

<p align="center">
  <a href="https://github.com/Gentleman-Programming/gentle-ai"><img alt="Built with Gentle-AI" src="https://raw.githubusercontent.com/Gentleman-Programming/gentle-ai/main/docs/assets/brand/built-with-gentle-ai.png"></a>
</p>

Task tracking for [Claude Code](https://claude.com/claude-code): `/task:*`
commands, local timers, and an event dispatcher that hands pipeline events
to whichever task tracker provider you have installed (Teamwork today).
It installs and works on a machine with no [nerv](https://github.com/war-apps/nerv-by-gentle-ai)
at all.

## Requirements

- The [`claude` CLI](https://code.claude.com/docs) on your `PATH`:
  `nerv-task install` registers the marketplace and plugins through it.
- `install.sh` (Linux, macOS): `curl`, `tar`, and `sha256sum` or `shasum`.
- `install.ps1` (Windows): PowerShell 5.1 or later.
- Release binaries: Linux, macOS and Windows, on amd64 and arm64.
- Building from source: Go 1.26.

## Install

**Linux / macOS:**

```sh
curl -fsSL https://raw.githubusercontent.com/war-apps/nerv-task-by-gentle-ai/main/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/war-apps/nerv-task-by-gentle-ai/main/scripts/install.ps1 | iex
```

Either script downloads the right release for your platform, verifies it
against the release's `checksums.txt` (SHA-256), installs the
`nerv-task` binary, and runs `nerv-task install`, which materializes and
registers the `nerv-task` plugin marketplace in Claude Code.

The binary goes to `~/.local/bin/nerv-task` (`install.sh`, which warns
when that folder is not on your `PATH`) or
`%LOCALAPPDATA%\nerv-task\bin\nerv-task.exe` (`install.ps1`, which adds
it to your user `PATH`). The scripts take no arguments; set these
environment variables instead:

| Variable | Effect |
|---|---|
| `NERV_TASK_VERSION` | Exact tag to install (for example `v1.2.0`); default: the latest stable release |
| `NERV_TASK_INSTALL_DIR` | Where to put the binary |
| `NERV_TASK_NO_INSTALL` | Any non-empty value: place the binary, skip `nerv-task install` |
| `NERV_TASK_TTY_DEVICE` | `install.sh` only: a tty (such as `/dev/tty`) to reattach as stdin, so `nerv-task install` can prompt even when piped from `curl` |

When `install.sh` runs with no terminal on stdin (and no usable
`NERV_TASK_TTY_DEVICE`), it runs `nerv-task install --no-configure`.

If `nerv-task` is already on your `PATH`, you can also just run:

```sh
nerv-task install
```

Add `--with-teamwork` to also enable the Teamwork provider plugin:

```sh
nerv-task install --with-teamwork
```

Without the flag, install still enables `task-teamwork@nerv-task` when
nerv's own `task-teamwork@nerv` was enabled, so the Teamwork provider is
never lost in the switch.

If nerv's own `task@nerv` or `task-teamwork@nerv` plugin is enabled, install
disables it (never touching any other nerv plugin), so `/task:*` is never
duplicated between the two marketplaces.

Install writes the marketplace to `~/.nerv-task/marketplace`, registers
it and enables its plugins (`task@nerv-task`, plus
`task-teamwork@nerv-task` when selected) in `~/.claude/settings.json`,
and installs them through the `claude` CLI (recorded in
`~/.claude/plugins/installed_plugins.json`). Restart Claude Code
afterwards.

### Upgrade

```sh
nerv-task upgrade [--alpha|--rc]
```

Replaces the binary with the newest stable release (or the newest alpha
or release candidate), then re-runs install, which refreshes every
enabled `<name>@nerv-task` plugin. To pin a version instead, re-run an
install script with `NERV_TASK_VERSION` set. `nerv-task version --json`
prints `{"binary": "...", "plugin": "..."}`; this is what nerv reads.

### Uninstall

There is no uninstall command. By hand:

```sh
claude plugin uninstall task@nerv-task
claude plugin uninstall task-teamwork@nerv-task   # if installed
claude plugin marketplace remove nerv-task
```

Then delete `~/.nerv-task/` and the `nerv-task` binary. Your
`task.yaml` files and the timer store (`~/.claude/work/timers.json`)
stay until you remove them.

## Commands

| Command | What it does |
|---|---|
| `/task:me`, `/task:all` | List pending tasks, with running timers and tracked time |
| `/task:start` | Start working on a task: timer, assignment, stage, worktree offer |
| `/task:stop`, `/task:stopAll` | Stop tracking and log time |
| `/task:done`, `/task:close` | Mark implemented, or close (complete) a task |
| `/task:blocked`, `/task:canceled` | Block or cancel a task, with its mandatory cause |
| `/task:analysis` | Move a task to the analysis stage and start a local timer on it |
| `/task:pending`, `/task:testing` | Move a task between stages |
| `/task:take` | Assign a task to yourself, without starting anything |
| `/task:priority` | Change a task's priority |
| `/task:timers`, `/task:clear` | Inspect, or discard without logging, a local timer |
| `/task:setup` | Configure this repository's task tracking, interactively |
| `/task:replay` | Re-dispatch task events a run could not deliver |

Full contract: [`plugin/task/skills/task/SKILL.md`](plugin/task/skills/task/SKILL.md).

## CLI

| Command | What it does |
|---|---|
| `nerv-task install [--with-teamwork] [--no-configure]` | Materialize and register the plugin marketplace; on a terminal, offers to run `nerv-task configure` once installed |
| `nerv-task configure [--print] [--json] [--set key=value] [--answers <file>]` | On a terminal with no flags, runs an interactive wizard: pick which providers to enable, their personal settings, and the timer defaults, with a diff and a confirmation before writing. Without a terminal, prints the current configuration and how to set a key. `--answers <file>` drives the wizard from a file (one answer per line) instead of a real terminal. |
| `nerv-task init [--provider <name>] [--import] [--none]` | Write this repository's own `task.yaml` |
| `nerv-task upgrade [--alpha\|--rc]` | Replace the binary with the newest release, then reinstall |
| `nerv-task version [--json]` | Print the binary and task plugin versions |

## Configuration

Two files, both YAML (see `plugin/task/skills/task/SKILL.md` §1 for the
full key reference). The repository file wins when both set a key.

- **Repository** — `<repo>/.ai-workflow/task.yaml`: the keys below, plus
  every provider setting whose manifest marks it `scope: repo` (such as a
  project or tasklist id).

  | Key | Default | Meaning |
  |---|---|---|
  | `provider` | the only installed provider | Provider for bare task ids and listings; `none` turns task tracking off here |
  | `ask_when_missing` | `true` | Ask for a task when a run starts without one |
  | `subtasks_per_wave` | `false` | Create one subtask per nerv FULL wave |
  | `events.<event>` | `on` | Per-event switch: `on` or `off` |
  | `sources` | none | Extra listing sources folded into `/task:me` and `/task:all` |
  | `providers.<name>.enabled` | `true` | `false` keeps the provider installed but sends it no events |

- **Global** — `<config dir>/ai-workflow/task.yaml`, where `<config dir>`
  is `$XDG_CONFIG_HOME` (default `~/.config`) on Linux,
  `~/Library/Application Support` on macOS and `%AppData%` on Windows;
  `nerv-task configure --print` shows the path in use. It holds the
  machine-scoped `timer_store`, `stale_after_hours` and the older
  `rounding_minutes`, every provider setting marked `scope: user` (such
  as your own assignee id, set with `nerv-task configure --set
  <provider>.<key>=<value>`), plus a `defaults:` block that seeds a new
  repository's file.

`nerv-task init` writes the repository file for you, seeded from your
global `defaults:`. `nerv-task configure` edits the global file,
interactively on a terminal.

### Timers

Timers are always local, never tracker-side, so several tasks and
sessions can run in parallel. They live in `timer_store` (default
`~/.claude/work/timers.json`). A timer older than `stale_after_hours`
(default `12`) is stale: nerv-task proposes a cut-off from your last
activity instead of logging it up to now. Each logged block is rounded
up to the provider's `rounding_minutes` (Teamwork default: `15`).

### Providers

Teamwork (`task-teamwork`) is the only built-in provider. A repository can
ship its own as `.ai-workflow/providers/task-<name>/provider.yaml`,
committed in the main checkout; the manifest and its operations are
specified in [`SKILL.md` §2–§3](plugin/task/skills/task/SKILL.md#2-finding-providers).

### Teamwork

Connect it through one of: the claude.ai Teamwork connector, the
plugin's own MCP server (signed in once with `/mcp`, OAuth), or a server
you register by hand with a bearer token. Deleting a timelog uses
Teamwork's REST API and needs `TEAMWORK_API_KEY`. Details:
[`plugin/extensions/task-teamwork/README.md`](plugin/extensions/task-teamwork/README.md).

Its keys live under `providers.teamwork.*`:

- global: `assignee_id`, `default_project_id`, `default_tasklist_id`,
  `rounding_minutes`, `billable`, `stages`, `stage_variants`,
  `comment_labels`, `comment_rest`, `known_projects`;
- repository: `project_id`, `tasklist_id`.

Stages match case- and accent-insensitive substrings of your board's
stage names, and the defaults are Spanish (`PENDIENTE`, `ANALISIS`,
`DESARROLLO`, `TESTING`, `IMPLEMENTA`, `BLOQUEA`, `CANCEL`). If your
board uses other names, override them under `providers.teamwork.stages`.

### Git settings

The `/task:start` worktree offer uses `branch_pattern`, `worktree`
(`ask`, `always` or `never`), `worktree_pattern`, `base_branch` and
`commit_ref_pattern`. They default to nerv's own documented defaults.
When nerv is set up, nerv-task reads the `git:` block of nerv's global
`config.yaml` (next to the global `task.yaml`), then of the repository's
`.ai-workflow/config.yaml`, which wins key by key.

## Relation to nerv

nerv-task is fully independent: install it on its own and it just works.
Without nerv there are no events; you drive everything with `/task:*`.
If you also use [nerv](https://github.com/war-apps/nerv-by-gentle-ai), it
can optionally emit pipeline events to nerv-task's `task:dispatcher`
agent — nerv-task declares this contract in `plugin/task/nerv-events.yaml`,
with no dependency in the other direction:

| Event | The dispatcher |
|---|---|
| `run.started` | Assigns the task, moves it to in development, starts a timer |
| `verification.started` | Moves the task to testing |
| `audit.ranked` | Comments the ranked audit summary |
| `issue.deferred` | Creates one task per deferred issue |
| `wave.started` | Creates a subtask, only when `subtasks_per_wave` is `true` |
| `run.closed` | Stops the timer and moves the task to implemented; never completes it (use `/task:close`) |
| `run.halted` | Comments the cause, moves the task to blocked, stops the timer |

Dispatch is fail-soft and never retries: a failure never halts the run.
nerv records undelivered events, and `/task:replay` re-sends them.

## Development

```sh
go install github.com/war-apps/nerv-task-by-gentle-ai/cmd/nerv-task@latest
go test ./...
```

Requires Go 1.26. Changes: [CHANGELOG.md](CHANGELOG.md). Releases:
[github.com/war-apps/nerv-task-by-gentle-ai/releases](https://github.com/war-apps/nerv-task-by-gentle-ai/releases).

## License

[CC0 1.0](LICENSE).
