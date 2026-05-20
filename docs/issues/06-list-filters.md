# 06 — List + category/tag filters

**Type:** AFK
**Parent PRD:** `docs/prd/v1-thought-box.md`

## What to build

A paginated browsing surface. `GET /thoughts` returns the user's thoughts in reverse chronological order with cursor pagination and optional `category` and `tag` filters. The PWA's list view renders thoughts (showing title, summary, timestamp, category, tags), supports filter chips for category and tag, and continues to poll any pending rows so they animate into completion in place.

After this slice, the user has a working "go back to my thoughts" surface — the second half of the product loop.

## Acceptance criteria

- [ ] `GET /thoughts?limit=&before=&category=&tag=` returns `{ items: [...], next_cursor: ... }`.
- [ ] Cursor pagination keyed by `created_at` (newest first; `before=<timestamp>` returns rows older than the cursor). Default limit 50, max 100.
- [ ] `category` filter: when present, only thoughts whose enrichment matches the given category are returned.
- [ ] `tag` filter: when present, only thoughts whose enrichment tags array contains the given tag are returned (`'kotlin' = ANY(tags)` semantics).
- [ ] Multiple filters compose with AND.
- [ ] Thoughts without enrichment yet (status `pending`/`transcribing`/`enriching`) are returned with null `enrichment`; they are excluded when a `category` or `tag` filter is active.
- [ ] All queries are scoped to `current_user_id` from `UserResolver`.
- [ ] PWA list view fetches the first page on mount; supports loading more via the cursor.
- [ ] PWA filter chips: category chips (the closed enum) and a freeform tag chip input. Selecting a filter triggers a fresh fetch.
- [ ] PWA continues polling thoughts in non-terminal states even within the list view; their cards update in place when the pipeline completes.
- [ ] Unit tests: pagination cursor parsing (well-formed cursor accepted, malformed rejected); filter-parameter parsing.

## Blocked by

- Blocked by #05
