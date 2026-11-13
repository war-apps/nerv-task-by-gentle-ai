Call `twprojects-create_timelog`:
- `task_id`;
- `date` and `time` from the block's real start. `time` must be `HH:MM:SS`: the API rejects `HH:MM`;
- `hours` and `minutes` from `minutes` (`minutes` below 60);
- `billable` from the caller, default `billable`;
- `description`. Teamwork strips ` #NN` tokens, so rewrite any `#NN` as `PR NN` before sending.

Return `{timelogRef}`, the new timelog id.
