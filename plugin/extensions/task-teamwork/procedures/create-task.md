1. **Where.** Pick the tasklist in this order: the caller's `list` (an id, or a name resolved with `twprojects-list_tasklists` and its `search_term`), then `tasklist_id`, then `default_tasklist_id`. The project follows the same order (`project_id`, then `default_project_id`).
   Names in `known_projects` resolve as that map says: a superseded project only when it is named explicitly; a retired one only by explicit id.
2. **Create.** `twprojects-create_task` with `tasklist_id`, `name`, `description`, `priority` (default `high` for source rows) and no tags. It is unassigned: the caller assigns it next when it should be.
3. **Missing tasklist.** When no tasklist can be resolved, return `needs_input` asking for one, with the defaults as suggestions.

Return `{taskRef: "TW-<new id>"}`.
