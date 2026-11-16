1. **Parent.** `twprojects-get_task` on the parent, with `fields: [tasklist]`: the subtask goes in the parent's tasklist (Teamwork requires it).
2. **Create.** `twprojects-create_task` as `createTask` does, in that tasklist, with `parent_task_id` = the parent's id.

Return `{taskRef: "TW-<new id>"}`.
