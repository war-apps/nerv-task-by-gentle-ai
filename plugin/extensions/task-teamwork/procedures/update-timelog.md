Call `twprojects-update_timelog` with the timelog `id`, the new `hours` and `minutes`, and the `description`, with `#NN` rewritten as `PR NN`.

The start time stays where it is: when the caller needs a different start, it calls `logTime` for the replacement and `deleteTimelog` for the old timelog (skill §4.2 step 5).
