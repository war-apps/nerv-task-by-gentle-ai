# nerv-task

Task tracking for [Claude Code](https://claude.com/claude-code): `/task:*`
commands, local timers, and an event dispatcher that hands pipeline events
to whichever task tracker provider you have installed (Teamwork today).
It installs and works on a machine with no [nerv](https://github.com/war-apps/nerv-by-gentle-ai)
at all.

## Install

**Linux / macOS:**

```sh
curl -fsSL https://raw.githubusercontent.com/war-apps/nerv-task-by-gentle-ai/main/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/war-apps/nerv-task-by-gentle-ai/main/scripts/install.ps1 | iex
```

Either script downloads the right release for your platform, installs the
`nerv-task` binary, and runs `nerv-task install`, which materializes and
registers the `nerv-task` plugin marketplace in Claude Code.

If `nerv-task` is already on your `PATH`, you can also just run:

```sh
nerv-task install
```

Add `--with-teamwork` to also enable the Teamwork provider plugin:

```sh
nerv-task install --with-teamwork
```

If nerv's own `task@nerv` or `task-teamwork@nerv` plugin is enabled, install
disables it (never touching any other nerv plugin), so `/task:*` is never
duplicated between the two marketplaces.

## Commands

| Command | What it does |
|---|---|
| `/task:me`, `/task:all` | List pending tasks, with running timers and tracked time |
| `/task:start` | Start working on a task: timer, assignment, stage, worktree offer |
| `/task:stop`, `/task:stopAll` | Stop tracking and log time |
| `/task:done`, `/task:close` | Mark implemented, or close (complete) a task |
| `/task:blocked`, `/task:canceled` | Block or cancel a task, with its mandatory cause |
| `/task:analysis`, `/task:pending`, `/task:testing` | Move a task between stages |
| `/task:take` | Assign a task to yourself, without starting anything |
| `/task:priority` | Change a task's priority |
| `/task:timers`, `/task:clear` | Inspect, or discard without logging, a local timer |
| `/task:setup` | Configure this repository's task tracking, interactively |
| `/task:replay` | Re-dispatch task events a run could not deliver |

Full contract: `plugin/task/skills/task/SKILL.md`.

## CLI

| Command | What it does |
|---|---|
| `nerv-task install [--with-teamwork] [--no-configure]` | Materialize and register the plugin marketplace; on a terminal, offers to run `nerv-task configure` once installed |
| `nerv-task configure [--print] [--json] [--set key=value] [--answers <file>]` | On a terminal with no flags, runs an interactive wizard: pick which providers to enable, their personal settings, and the timer defaults, with a diff and a confirmation before writing. Without a terminal, prints the current configuration and how to set a key. `--answers <file>` drives the wizard from a file (one answer per line) instead of a real terminal. |
| `nerv-task init [--provider <name>] [--import] [--none]` | Write this repository's own `task.yaml` |
| `nerv-task upgrade [--alpha\|--rc]` | Replace the binary with the newest release, then reinstall |

## Configuration

Two files, both YAML (see `plugin/task/skills/task/SKILL.md` §1 for the
full key reference):

- **Repository** — `<repo>/.ai-workflow/task.yaml`: the provider, per-event
  switches, listing sources, and every provider setting whose manifest
  marks it `scope: repo` (such as a project or tasklist id).
- **Global** — `~/.config/ai-workflow/task.yaml` on Linux,
  `%AppData%\ai-workflow\task.yaml` on Windows: the machine-scoped settings
  (timer store, stale-timer threshold) and every provider setting marked
  `scope: user` (such as your own assignee id), plus a `defaults:` block
  that seeds a new repository's file.

`nerv-task init` writes the repository file for you, seeded from your
global `defaults:`. `nerv-task configure` edits the global file,
interactively on a terminal.

## Relation to nerv

nerv-task is fully independent: install it on its own and it just works.
If you also use [nerv](https://github.com/war-apps/nerv-by-gentle-ai), it
can optionally emit pipeline events (`run.started`, `verification.started`,
`audit.ranked`, `issue.deferred`, `wave.started`, `run.closed`,
`run.halted`) to nerv-task's `task:dispatcher` agent — nerv-task declares
this contract in `plugin/task/nerv-events.yaml`, with no dependency in the
other direction. nerv-task also opportunistically reads nerv's own
`.ai-workflow/config.yaml` `git:` block for its worktree offer, falling
back to its own built-in defaults when nerv is not set up.

## License

[CC0 1.0](LICENSE).
