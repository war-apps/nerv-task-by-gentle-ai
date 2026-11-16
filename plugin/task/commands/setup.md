---
description: Set up this repository's .ai-workflow/task.yaml interactively — the task provider, its repository keys and the connection check — without nerv-task configure's wizard
argument-hint: [provider]
---

# /task:setup

Build `<repo>/.ai-workflow/task.yaml`: pick the provider, propose its
repository keys, ask only what cannot be detected, write the file, and
check that the provider's tools are reachable. It is nerv-task's own
repository setup: it does not run `nerv-task configure`, `nerv-task init`
or any other part of a larger setup. Load the `task` skill first: its §1
(configuration) and §2 (finding providers, the tool check) are the source
of truth for every key and rule.

**Arguments.** `$ARGUMENTS` may be empty or one provider name (`teamwork`):
then that provider is picked without asking. Anything else: say what is not
accepted and stop.

Talk to the user in their language; everything written to files is in
English.

## 1. Preconditions

1. Run `git rev-parse --show-toplevel`; on failure, say the setup belongs
   to a git repository and stop. Work on the **main checkout** (the
   skill's §2), so a worktree configures the repository itself.
2. Read `.ai-workflow/task.yaml` when it exists.
   - It already has content: show it and ask **edit it** (the default) or
     **start over**. Stop and wait.
   - `provider: none`: say task tracking is off in this repository and
     ask whether to turn it on. Stop and wait.
3. Read the global `task.yaml` (the path `nerv-task configure --print
   --json` reports as `config_path`; without `nerv-task` on PATH, the
   skill's §1 paths). Hold:
   - `defaults`, the user's seed for a new repository file (same shape as
     the repository file), never applied at run time;
   - the personal keys it already sets under `providers.<name>.`.

## 2. Provider

List the task providers the skill's §2 finds: the installed ones
(`task-<name>@nerv-task` in `~/.claude/plugins/installed_plugins.json`)
and the repository's own
(`.ai-workflow/providers/task-<name>/provider.yaml`), each with its
`name`, its `task_ref_prefix` and where it comes from.

- `$ARGUMENTS` names a provider that is not in the list: say it is not
  installed, list the ones that are, give the install line
  `nerv-task install --with task-teamwork`, and stop. Never fall back to
  another provider.
- None: say no task provider is installed, give the install line
  `nerv-task install --with task-teamwork`, and stop.
- The one `$ARGUMENTS` names: pick it.
- One: propose it.
- Several: the one `defaults.provider` names, else ask which one. Stop
  and wait.

## 3. Propose the file

Read the chosen provider's `provider.yaml` `config` block and show the
whole repository file as one YAML block:

- `provider: <name>`.
- `providers.<name>.<key>` for each `scope: repo` key the provider
  declares, with its `prompt` as a comment. Leave out a key whose value
  equals its `default`.
- **Seed.** Copy each key of `defaults` that the file does not set yet,
  once (the repository's values always win), leaving out any `scope:
  user` key. Mark every copied key in the proposal ("from your
  `defaults`").
- `ask_when_missing`, `subtasks_per_wave`, `events` and `sources` only when
  the user asks for other values than the skill's §1 defaults.

`scope: user` keys (such as `assignee_id`) never go into the repository:
they belong in the global `task.yaml` under `providers.<name>.`. List the
ones the global file does not set yet, with their `prompt`, and say
`nerv-task configure` edits them.

## 4. Ask only what cannot be detected

Ask one question at a time, with a recommended default, then STOP and wait.
When the provider's tools are reachable (§6), offer its own listings to
answer them: for `teamwork`, `twprojects-list_projects` for `project_id`
and `twprojects-list_tasklists` of that project for `tasklist_id`. Read
them only; create nothing in the provider.

Show the final file in full and ask once whether to write it. Write only
on an explicit yes.

## 5. Write

- No `.ai-workflow/task.yaml`: create it with the proposed content.
- Otherwise rewrite the whole file. Then read it back and check that it
  parses as YAML and holds what was shown. On any difference, restore the
  previous content and say what went wrong.

## 6. Connection

Run the skill's tool check (§2) for the chosen provider against the tools
available to this session.

- Every tool reachable: say through which source (a server, the
  provider's own plugin, a claude.ai connector).
- Missing tools: name them, and when the provider's root has a `README.md`
  (`task-teamwork` does), point to its connection section and summarize its
  options in one line each. Do not change any MCP configuration.

## 7. Next steps

Tell the user, without doing any of it: set the listed `scope: user` keys
with `nerv-task configure`, commit `.ai-workflow/task.yaml` when ready,
then run `/task:me`.

Never commit, push, create or change anything in the task provider, or
edit any file other than `.ai-workflow/task.yaml`.
