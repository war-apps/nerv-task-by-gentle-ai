# nerv-task: the task tool of the NERV ecosystem, independent of nerv

Objective: move nerv's task extension (`plugins/task`, `plugins/task-teamwork`, `internal/taskprovider` and the task-only configure/wizard code) into this repository as an independent tool with its own CLI, install scripts, releases and configuration, that installs and works on a machine without nerv.
Master plan: `war-apps/nerv-by-gentle-ai` → `odd/tasks/ecosystem-split.md`, phase P2 (specs S2-S5, S8, S9).
Branch: `feat/initial-task` from `develop`. Module `github.com/war-apps/nerv-task-by-gentle-ai`, Go 1.26, binary `nerv-task`. Depends on nerv-kit v0.1.1. Releases start at 1.0.0. TDD: strict (session configuration); runner `go test ./...`.
Source to port (read-only): nerv repo `develop` at 93982c0.

## Specs

- **T-S1** Plugins: one marketplace `nerv-task` with two plugins, `task` (commands `/task:*`, agent `task:dispatcher`, skill `task`) and `task-teamwork` (provider, procedures, `.mcp.json`). The `task-teamwork` name is load-bearing: MCP tools resolve as `mcp__plugin_task-teamwork_teamwork__*`.
- **T-S2** Configuration (kit `aiworkflow`): repo `.ai-workflow/task.yaml` with today's `tasks.*` keys minus the `tasks.` prefix (`provider`, `ask_when_missing`, `subtasks_per_wave`, `events`, `sources`, `providers.<p>.*` with `scope: repo`, `providers.<p>.enabled`); global `~/.config/ai-workflow/task.yaml` (`%AppData%` on Windows) with `timer_store`, `stale_after_hours`, `rounding_minutes` (legacy), `providers.<p>.*` with `scope: user`, and `defaults:` (seed for `init`). A provider's `config[].scope` decides the file.
- **T-S3** Plugin text: every reference to `.ai-workflow/config.yaml` `tasks:`, `tasks.providers.*`, `defaults.tasks`, `nerv configure` or `nerv install` becomes `task.yaml`, `providers.*`, `defaults:`, `nerv-task configure`, `nerv-task install`.
- **T-S4** Event contract: the `task` plugin ships `nerv-events.yaml` at its root (`agent: task:dispatcher`, `events: [run.started, verification.started, audit.ranked, issue.deferred, wave.started, run.closed, run.halted]`), no "extension" wording. nerv (P3) finds subscribers through it.
- **T-S5** Git settings for the worktree offer (`branch_pattern`, `commit_ref_pattern`, `worktree`, `worktree_pattern`, `base_branch`): built-in defaults equal to nerv's documented ones; read nerv's `.ai-workflow/config.yaml` `git:` block opportunistically when present; never a dependency.
- **T-S6** Provider contract: a trimmed port of nerv `internal/taskprovider` plus the provider-manifest slice of `internal/extension` (`ParseProvider`, `LoadProvider`, `CheckAgainst`, config keys, MCP, statuses, `ConnectorToolPrefix`); no exec runtime, slots, requires or suggestion order. Discovery: embedded plugin tree and `.ai-workflow/providers/task-*/provider.yaml`.
- **T-S7** CLI: `nerv-task install` (materialize and register marketplace `nerv-task` with kit `claudeplugin`, offer `task-teamwork`, offer the global configuration; when `task@nerv` or `task-teamwork@nerv` is enabled, disable them with a message so `/task:*` is not duplicated; never touches other nerv plugins), `nerv-task upgrade [--alpha|--rc]` (kit `selfupgrade`, then reinstall), `nerv-task configure [--print] [--json] [--set k=v]` (global defaults and provider choice), `nerv-task init` (repository `task.yaml`, "none" allowed; when `.ai-workflow/config.yaml` has a `tasks:` block, offer to import it).
- **T-S8** Packaging: goreleaser (linux/darwin/windows, amd64/arm64, Windows zip with `nerv-task.exe`, LICENSE in archives); `scripts/install.sh` and `scripts/install.ps1` (user decision 2026-11-13: one-command install on Windows too); CI on Ubuntu and Windows (gofmt, vet, test) plus a job that installs `nerv-task` on a runner with no nerv and checks the plugin registration; release workflow publishing stable releases from `main`.
- **T-S9** Tests: port `task_setup_prose_test`, `teamwork_stage_comment_prose_test`, the `TestTaskEnvelope_ResultListsMatchContract` and `TestTaskContract_TestingMoveFollowsWorkUnitVerification` checks, and a dispatcher check that names every event; provider contract unit tests from nerv.
- **T-S10** Before publishing: scan docs and examples for internal hostnames or company data (master S9).

## Tasks

| ID | Specs | Route | Status | Commit |
|---|---|---|---|---|
| W1 | T-S6 | delegated writer | done | 42ab15e |
| W2 | T-S1, T-S3, T-S4, T-S9 | delegated writer | done | d538092 |
| W3 | T-S2, T-S7 (install, configure) | delegated writer | done; interactive configure added after review | fff8035, 668f7bd |
| W4 | T-S2, T-S5, T-S7 (init) | delegated writer | done; security fix: never import personal keys from a repository into the global config (amended); git defaults fixed after review | b143ea2, 0902a9e |
| W5 | T-S7 (upgrade), T-S8, T-S10 | delegated writer | done | 6aaf5d4, c71a932 |

## Log

- **L1** (2026-11-13) Design from the P2 research; user confirmed `install.ps1`. Defaults taken: `nerv-events.yaml`, opportunistic git settings, import offer on `init`.
- **L2** (2026-11-13) Security review: `init --import` copied a repository's `scope: user` provider keys into the global task.yaml (cross-repository config poisoning); fixed with a RED test: personal keys from a repository are never written, only reported by name; imported repo keys are type-checked.
- **L3** (2026-11-16) Parent review: the writer's git defaults were guesses (4 of 5 wrong, `worktree` a bool); fixed to nerv's catalogue values (`develop`, `ask`, `.claude/worktrees/{slug}`, `feature/{prefix}-{id}-{slug}`, `({PREFIX}-{id})`), reading nerv's user config then the repo's (0902a9e). Interactive `configure` (provider choice, user keys with defaults, diff and confirm; offered at the end of `install`; `--answers` for tests) via golang.org/x/term like nerv (668f7bd).
- **L4** (2026-11-16) Verification: gofmt/vet/build clean, `go test -count=1 ./...` 8 packages ok on Windows, `go mod tidy` no-op. Sandbox run of the real binary with an isolated HOME holding `nerv@nerv`, `task@nerv`, `task-teamwork@nerv`, `release@nerv`: install disabled only the two task plugins, kept `nerv@nerv` and `release@nerv`, registered marketplace `nerv-task` and enabled `task@nerv-task`, `task-teamwork@nerv-task`; the real ~/.claude was untouched.
- **L5** (2026-11-16) PR #1 CI: Linux tests read the runner's `XDG_CONFIG_HOME` instead of the fake home (TestMain clears it in internal/config and internal/configure); the Windows runner checked out CRLF and broke a prose test (`.gitattributes` keeps LF everywhere, so embedded plugin bytes are identical on every OS); matrix `fail-fast: false`. Ubuntu, Windows and install-without-nerv green.
