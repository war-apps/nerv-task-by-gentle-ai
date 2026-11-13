1. **Check.** `twprojects-get_task` with `fields: [assignees]`. When `assignee_id` is already among the assigned users, return `{assigned: false}`.
2. **Assign.** Otherwise call `twprojects-update_task` with `assignees.user_ids` = the current assigned users plus `assignee_id`, keeping any `company_ids`, `team_ids` and `job_role_ids` it had. Never drop an existing assignee.

When `assignee_id` is unset, the result is `blocked` with `providers.teamwork.assignee_id is not set`.
