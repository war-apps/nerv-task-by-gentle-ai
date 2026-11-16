Call `twprojects-list_timelogs`:
- with `task_id`, or with `project_id` and `assigned_user_ids: [assignee_id]` for a listing. Use one call per project, never one per task;
- with `fields: [id, description, minutes, timeLogged, task]`, and `start_date`/`end_date` when the caller gives a window.

Return rows `{timelogRef, taskRef, start, minutes, description}`.
