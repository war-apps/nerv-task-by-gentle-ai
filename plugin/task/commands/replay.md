---
description: Re-dispatch the task events a run could not deliver (blocked or failed with replay: true)
argument-hint: <feature or run id>
---

1. **Read.** Argument is the run summary's hint: a FULL run's feature, or a DIRECT/LIGHT run's run id. Read `odd/{feature}/nerv/task-events.yaml` (FULL) or `.nerv/runs/{run-id}/task-events.yaml` (DIRECT/LIGHT). When neither exists, list features/run ids with pending events, then stop.
2. **Show.** Show pending envelopes: event, provider, error.
3. **Confirm.** Ask once whether to replay them all.
4. **Replay.** On yes, dispatch each event again, in order, with its original payload. Use the `task:dispatcher` agent, or run its steps inline in this session so it can answer `needs_input` questions.
5. **Record.** Mark each envelope `replayed: <ISO time>` with its new result.

This command's own result — one line per envelope with its old and new results — is shown directly in the session.
