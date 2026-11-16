# task-teamwork

The Teamwork provider for `nerv-task`: it assigns, moves stages, logs time and
creates tasks in [Teamwork](https://www.teamwork.com), using Teamwork's
official MCP server ([github.com/Teamwork/mcp](https://github.com/Teamwork/mcp)).

## Connect to Teamwork

Three sources can serve the MCP tools this provider calls, and `nerv-task`
picks whichever one is actually available, in this order:

| Source | Sign-in | Claude Code tool names |
|---|---|---|
| A. claude.ai Teamwork connector | claude.ai, once | `mcp__claude_ai_Teamwork_com__twprojects-*` |
| B. This plugin's own MCP server (OAuth) | per machine | `mcp__plugin_task-teamwork_teamwork__twprojects-*` |
| C. Fixed bearer token via environment variable | none | `mcp__teamwork__twprojects-*` |

For B, this plugin ships the server in its `.mcp.json`; sign in once with
`/mcp` in Claude Code. For C, register the server by hand:

```
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com \
  --header "Authorization: Bearer <token>"
```

`deleteTimelog` has no MCP tool and uses Teamwork's REST API directly
(`DELETE /projects/api/v3/time/{timelogId}.json`), authenticated with the
`TEAMWORK_API_KEY` environment variable.

## Configuration

Set these under `providers.teamwork.*` in your task configuration (see the
repository root README for where `task.yaml` lives):

```yaml
providers:
  teamwork:
    assignee_id: 100          # your Teamwork user id (global task.yaml)
    default_project_id: 1     # default project for new tasks (global task.yaml)
    default_tasklist_id: 2    # default tasklist for new tasks (global task.yaml)
    project_id: 1             # this repository's project (repo task.yaml)
    tasklist_id: 2            # this repository's tasklist (repo task.yaml)
    rounding_minutes: 15
    billable: false
```

`/task:setup` and `nerv-task configure` walk through these interactively.

## Stages

The provider matches canonical stage keys (`pending`, `analysis`, `inDev`,
`testing`, `implemented`, `blocked`, `canceled`) against case- and
accent-insensitive substrings of your board's real stage names. The
defaults are tuned for a Spanish-language board; override any of them
under `providers.teamwork.stages.<key>`. A task with no workflow at all
gets its stage recorded as a comment instead (`ESTADO: <stage>`).
