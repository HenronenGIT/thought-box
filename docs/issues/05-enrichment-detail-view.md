# 05 — Enrichment + detail view

**Type:** AFK
**Parent PRD:** `docs/prd/v1-thought-box.md`

## What to build

Add the LLM enrichment stage after transcription. An `Enricher` interface with an `OpenAiEnricher` implementation calls chat completions with JSON-schema-enforced structured outputs, returning a category (from a fixed enum), tags (open-ended list), generated title, and one-sentence summary. The `ThoughtPipeline` is extended to drive `done` thoughts through an `enriching` stage. Enrichment rows live in their own `thought_enrichments` table linked to `thoughts`. The PWA's detail view renders the enrichment alongside the transcript and audio.

This slice closes the v1 capture loop: after speaking, the user sees a titled, summarized, categorized, tagged thought.

## Acceptance criteria

- [ ] `Enricher` interface defined: `suspend fun enrich(transcript: String): EnrichmentResult`.
- [ ] `EnrichmentResult` data class includes category, tags, title, summary, model, prompt_version.
- [ ] `OpenAiEnricher` implementation calls chat completions with a JSON schema; uses `Idempotency-Key` header derived from `thought_id + prompt_version`.
- [ ] Closed category set defined as a Kotlin enum with at least 5 buckets (e.g., `idea`, `todo`, `feeling`, `question`, `observation`, `reminder`).
- [ ] Flyway migration creates `thought_enrichments` table: `thought_id` (uuid PK, FK with `ON DELETE CASCADE`), `category` (text with CHECK constraint matching enum values), `tags` (`text[]`, default empty), `title`, `summary`, `model`, `prompt_version`, `created_at`.
- [ ] State machine extended: after transcription succeeds, status flips to `enriching` (not `done`); on enrichment success, status flips to `done`; on retry-exhaustion, status flips to `failed_enrichment`.
- [ ] Retry policy reused from #04 (3 attempts, exponential backoff).
- [ ] `StartupRecovery` extended: rows stuck in `enriching` are reset to a recoverable state.
- [ ] `ThoughtPipeline` picks up rows ready for enrichment and runs them through `OpenAiEnricher`.
- [ ] `GET /thoughts/:id` returns the enrichment (when present) under an `enrichment` key alongside transcript and metadata.
- [ ] PWA detail view renders title, summary, category, and tags from the enrichment.
- [ ] PWA renders a clear failure state when status is `failed_enrichment`.
- [ ] Unit tests: `EnrichmentResult` deserialization (well-formed JSON, malformed JSON, refusal-shape response); state machine transitions for the extended pipeline.

## Blocked by

- Blocked by #04
