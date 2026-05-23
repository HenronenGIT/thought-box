# Echo runs as a decoupled sweeper, not part of the Thought pipeline

The Thought pipeline (`pending → transcribing → enriching → done`) does not know about Echoes. A separate worker sweeps `thoughts WHERE status = 'done' AND no default echo exists` and drives Echo generation through its own status field on a separate `echoes` table.

We chose this over (a) extending the Thought pipeline with an `echoing` state and (b) a `LISTEN/NOTIFY` push trigger.

## Why

- **Echoes are supplementary.** A failed Echo must not make a Thought look broken. Decoupling failure domains means the Thought reaches `done` and is readable regardless of what happens to its Echo.
- **Echo is deletable in one commit.** Because the main pipeline never references the `echoes` table, removing the feature (or rebuilding it differently) does not touch the Thought lifecycle.
- **The sweeper pattern already exists in the codebase** (`StartupRecovery`). Reusing it keeps the mental model small.
- **`LISTEN/NOTIFY` was rejected** because it would introduce a new infrastructure pattern (reconnect handling, listener lifetime) used nowhere else, in exchange for shaving a few seconds off a non-critical-path operation.

## Consequences

- Echo gen lags Thought `done` by up to one sweep interval. Acceptable — Echo is not part of the capture-to-read latency budget.
- A partial unique index on `(thought_id, mode)` excluding `failed` rows is required to prevent sweeper races from inserting duplicate echoes.
- The "echo is missing" state is indistinguishable to the user from "echo is still generating" (both render nothing per the silent-failure UX). That's intentional: the user should not be aware of Echo's lifecycle at all.
