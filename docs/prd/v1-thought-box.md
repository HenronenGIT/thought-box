# PRD — Thought Box v1

## Problem Statement

People have ideas constantly — on walks, between meetings, in the middle of unrelated tasks — and the act of writing them down breaks the moment. Notes apps assume the user can stop, type, and structure their thought. By the time a thought is typed, it has often been edited, abbreviated, or lost. There is no low-friction surface that captures a half-formed thought as fast as it arrives and turns it into something the user can come back to later, search through, and reason about as the collection grows.

## Solution

A thought-collector application with a dictation-first capture surface. The user taps a button, speaks their thought (up to one minute for v1), and the system handles the rest: it persists the original audio, transcribes the speech to text, and enriches each thought with a generated title, a one-sentence summary, a closed-set category, and open-ended tags. The user can browse their thoughts in reverse chronological order, filter by category or tag, replay the original audio, and read the transcript and summary.

The backend is written in Kotlin so the user can learn the language, the JVM, and the surrounding ecosystem. The frontend is a Progressive Web App so capture works anywhere the user has a browser, without the friction of installing a native app.

## User Stories

1. As a user with a fleeting idea, I want to start recording in one tap from the PWA so that I can capture the thought before it slips away.
2. As a user, I want the recorder to use my browser's microphone and stop automatically when I tap stop, so that I do not have to fiddle with controls while speaking.
3. As a user, I want a visible recording indicator and elapsed-time display so that I know the recording is active and how much time I have left.
4. As a user, I want the recorder to stop automatically at the configured maximum duration so that I do not accidentally produce an over-long clip.
5. As a user on Safari, I want recording to work even though Safari does not support WebM, so that I can use the app on iOS.
6. As a user, I want the recorder to refuse to submit clips shorter than the configured minimum so that I do not waste a transcription on an accidental tap.
7. As a user, I want my recorded audio uploaded to the backend as soon as I stop, so that I can move on to the next thing.
8. As a user, I want the original audio preserved so that I can re-listen to a thought later in my own voice.
9. As a user, I want the transcription to appear shortly after I finish recording, so that I can verify the system captured my thought correctly.
10. As a user, I want the transcript displayed even before enrichment finishes, so that the most important content is available as soon as possible.
11. As a user, I want each thought to receive a short generated title so that scrolling a list of thoughts is meaningful rather than a wall of timestamps.
12. As a user, I want each thought to receive a one-sentence summary so that I can scan many thoughts quickly.
13. As a user, I want each thought tagged with a category (from a fixed set) so that I can filter by the kind of thought.
14. As a user, I want each thought tagged with open-ended tags so that I can find thoughts about a specific subject I care about.
15. As a user, I want to browse my thoughts in reverse chronological order (newest first) so that recent thoughts are immediately visible.
16. As a user, I want pagination on the list so that loading many thoughts stays fast.
17. As a user, I want to filter the list by category so that I can see only my todos or only my ideas.
18. As a user, I want to filter the list by tag so that I can find every thought about a given topic.
19. As a user, I want to open a single thought and see the audio, transcript, title, summary, category, and tags together so that I have full context.
20. As a user, I want to replay the original audio from the thought detail view so that I can hear the tone and inflection of the original.
21. As a user, I want the PWA to keep working even after I leave it idle so that the first request after a pause is not noticeably slow.
22. As a user, I want failed transcriptions or failed enrichments retried automatically a few times so that transient API errors do not strand my thought.
23. As a user, I want a clearly visible failure state on the thought if every retry fails so that I know something went wrong and can act on it.
24. As a user, I want the app to come back gracefully if the backend restarts mid-pipeline so that no thought is left silently stuck.
25. As a future user (multi-user mode), I want my thoughts to belong to me and be inaccessible to other users so that my private notes stay private. (v1: schema is in place; access control deferred.)
26. As the developer, I want to run the entire stack on my laptop against a local Postgres and local S3 emulator (MinIO) so that I can develop offline without touching production data.
27. As the developer, I want strict separation between dev and prod environments (separate buckets, separate databases, separate OpenAI API keys) so that I cannot accidentally write to production from my laptop.
28. As the developer, I want to log out which environment the application booted into at startup so that I can verify my configuration before it does damage.
29. As the developer, I want the backend to expose its current limits (max duration, max size) via an HTTP endpoint so that the PWA can enforce them without redeploying when the limits change.
30. As the developer, I want unhandled exceptions reported to Sentry so that I am notified of production errors without having to tail logs.
31. As the developer, I want structured JSON logs to stdout so that I can grep them with `jq` and ship them anywhere later without re-instrumenting.
32. As the developer, I want a CI job that builds and tests every push so that I never deploy code that does not compile.
33. As the developer, I want deploys triggered manually from GitHub Actions so that pushing to `main` does not automatically ship to production.
34. As the developer, I want the production backend reachable over HTTPS with a valid certificate so that the PWA can call it without mixed-content errors.
35. As the developer, I want to swap LLM and STT providers in the future by writing one new implementation class so that vendor lock-in does not block experimentation.
36. As the developer, I want to write raw SQL (not an ORM) so that I learn Postgres deeply and stay in control of every query.
37. As the developer, I want database schema changes versioned and applied by Flyway on startup so that local and production schemas never drift.

## Implementation Decisions

### Architecture and hosting

- **Frontend:** Next.js (App Router) deployed on Vercel free tier. Static + serverless functions as needed.
- **Backend:** Kotlin + Ktor, packaged as a fat JAR via the Shadow plugin, containerized with a multi-stage Dockerfile (`gradle:8-jdk21` for build, `eclipse-temurin:21-jre-alpine` for runtime), deployed to Google Cloud Run with scale-to-zero. Cloud Run provides HTTPS automatically at a `*.run.app` URL.
- **JDK:** 21 (Temurin LTS).
- **Build:** Gradle with Kotlin DSL, version catalog (`libs.versions.toml`).
- **Database:** Neon serverless Postgres (free tier). Accessed via raw SQL through kotliquery (no ORM). Schema migrations managed by Flyway, applied on application startup.
- **Audio storage:** AWS S3 (cross-cloud from Cloud Run, authenticated via IAM user keys passed as env vars).
- **STT:** OpenAI Whisper API.
- **LLM enrichment:** OpenAI `gpt-4o-mini` with JSON-schema-enforced structured outputs.
- **Cold-start mitigation:** PWA pings `GET /healthz` on mount to warm Cloud Run and the Neon Postgres compute.

### Modules

The backend is structured around deep, interface-driven modules so the orchestration code is small and the provider/storage details can be swapped without changing call sites.

- **`Config`** — Loads all runtime configuration from environment variables into a typed object at startup. Validates required values. Single source of truth; no other module reads env directly.
- **`Transcriber`** — Interface: `suspend fun transcribe(audio: AudioBlob): TranscriptionResult`. v1 implementation `OpenAiTranscriber` calls Whisper. Future providers implement the same interface.
- **`Enricher`** — Interface: `suspend fun enrich(transcript: String): EnrichmentResult`. v1 implementation `OpenAiEnricher` calls chat completions with a JSON schema. The `EnrichmentResult` includes category, tags, title, summary, plus metadata (model, prompt version). Uses an OpenAI `Idempotency-Key` derived from `thought_id + prompt_version`.
- **`BlobStore`** — Interface for `put`, `get`, `exists` over byte streams keyed by S3 object key. v1 implementation `S3BlobStore` wraps the AWS SDK. The same implementation talks to MinIO in dev by overriding the endpoint via env var.
- **`ThoughtRepository`** — kotliquery-backed data access. Insert, fetch by id, list with cursor pagination, status transitions, sweep queries for stuck rows. Returns domain types (`Thought`, `ThoughtEnrichment`), takes SQL inside.
- **`UserResolver`** — Resolves the "current user id" for a request. v1 returns a hardcoded seeded UUID. When auth lands, swap implementation to extract from JWT/session. Single resolver function used by all routes.
- **`ThoughtPipeline`** — Orchestrates the asynchronous transcription → enrichment flow. Reads `pending` rows, drives them through `transcribing` → `enriching` → `done`. Owns the retry policy (3 attempts, exponential backoff at 1s/4s/16s) and updates `attempts` / `last_error` columns. Runs in a dedicated `CoroutineScope` owned by the application.
- **`StartupRecovery`** — On application boot, sweeps any rows stuck in `transcribing` or `enriching` back to a recoverable state so the pipeline can pick them up cleanly.
- **HTTP routes** — Thin Ktor handlers. Parse requests, dispatch to repository/pipeline, render JSON. No business logic.

The frontend mirrors this with a small set of components: a recorder (MediaRecorder wrapper with MIME negotiation and a duration cap loaded from `/config`), a typed API client, a thought list view (with category/tag filters and polling for pending rows), a thought detail view (with audio playback), and a keepalive ping.

### API contract

Five HTTP endpoints. JSON over a small REST surface.

- `POST /thoughts` — multipart upload: audio blob + `mime_type` + `duration_ms` form fields. Streams audio to S3 (does not buffer fully in memory), inserts a `thoughts` row in `pending`, returns `{ id, status: "pending", created_at }`.
- `GET /thoughts/:id` — Returns the current state of a thought: status, transcript (if available), enrichment (if available), audio metadata, timestamps. Used by the client for polling.
- `GET /thoughts?limit=50&before=<cursor>&category=<cat>&tag=<tag>` — Cursor-paginated list, newest first. Optional filters for category and tag.
- `GET /thoughts/:id/audio` — Streams the original audio bytes back with the stored `Content-Type`.
- `GET /config` — Returns `{ max_duration_ms, min_duration_ms, max_size_bytes }`. Consumed by the PWA at load to drive client-side validation.
- `GET /healthz` — Lightweight liveness endpoint used by the PWA's keepalive ping.

### Pipeline and status model

A thought's lifecycle is a single `status` column with values `pending`, `transcribing`, `enriching`, `done`, `failed_transcription`, `failed_enrichment`. Transitions are owned by `ThoughtPipeline`. Failed terminal states populate `last_error` for diagnosis. Retries happen in-process via coroutines; there is no external job queue. After three exhausted retries the row enters a failed terminal state and is not retried automatically.

### Schema

Three tables.

- **`users`** — `id` (uuid PK), `created_at`. Intentionally minimal; auth-related columns added when auth lands. One seeded row for v1.
- **`thoughts`** — `id` (uuid PK), `user_id` (uuid FK), `created_at`, `updated_at`, `audio_s3_key`, `mime_type`, `duration_ms` (nullable until known), `size_bytes`, `transcript` (nullable), `status` (text with CHECK constraint), `attempts` (int, default 0), `last_error` (nullable), `last_attempt_at` (nullable), `transcribed_at` (nullable).
- **`thought_enrichments`** — `thought_id` (uuid PK, FK to `thoughts` with `ON DELETE CASCADE`), `category` (text with CHECK constraint, from a fixed enum), `tags` (`text[]`, default empty), `title`, `summary`, `model`, `prompt_version`, `created_at`.

Status is stored as text rather than as a Postgres enum type so future additions do not require `ALTER TYPE`. Tags use a Postgres array column to keep v1 simple; if tag-level metadata becomes valuable, this can be normalized later into a `tags` + `thought_tags` schema.

### Limits and configuration

- Maximum thought duration: 1 minute (configurable via `MAX_THOUGHT_DURATION_MS`).
- Minimum thought duration: 1 second (configurable via `MIN_THOUGHT_DURATION_MS`).
- Maximum file size: 10 MB (configurable via `MAX_THOUGHT_SIZE_BYTES`).
- Limits are enforced on both client (UX: countdown + auto-stop) and server (HTTP 400 on violation). Belt-and-suspenders because the client check is bypassable.
- The client fetches limits from `GET /config` on app load rather than hardcoding them. Changing a limit is an env-var change + restart, no PWA redeploy required.

### Environment isolation

Dev and production share no resources.

- Separate OpenAI API keys per environment, set per-environment via env vars.
- Separate S3 bucket names (`thoughts-dev` vs `thoughts-prod`) and DB names (`thoughts_dev` vs `thoughts_prod`).
- Local dev uses Docker Compose to run Postgres and MinIO. The backend's `BlobStore` accepts an `S3_ENDPOINT` override so the same code path talks to MinIO locally and S3 in production.
- The application logs the active environment (and the Postgres/S3 endpoints it has connected to) at startup.

### Observability

- Logback with `logstash-logback-encoder` produces JSON log lines to stdout. Cloud Run collects stdout automatically and exposes it in its console.
- Every request, every pipeline state transition, and every external API call logs a structured event. Transcript content and audio bytes are never logged.
- Sentry SDK for JVM wired into Ktor's exception handler. Free tier is sufficient for side-project volume.

### CI/CD

- GitHub Actions runs `./gradlew test shadowJar` on every push and PR. Fails on test failure or compile error.
- A separate `workflow_dispatch` job builds the Docker image, pushes to Google Artifact Registry, and runs `gcloud run deploy`. Triggered manually from the GitHub Actions UI.
- Frontend (Next.js) deploys to Vercel automatically on push to its production branch.
- Secrets (OpenAI keys, AWS IAM keys, Neon connection string, Sentry DSN, GCP service account) are stored as GitHub Actions secrets and as Cloud Run service environment variables.

### Provider abstraction

`Transcriber` and `Enricher` are interfaces. v1 has exactly one implementation of each (`OpenAiTranscriber`, `OpenAiEnricher`). Provider selection is wired in a single composition function — no runtime configuration, no factories, no plugin loading. Adding a second provider later is one new class plus one changed line in the wiring.

## Testing Decisions

### What makes a good test here

Tests verify externally observable behavior, not internal implementation. A `Transcriber` test passes the same audio in and checks the returned `TranscriptionResult` shape — it does not assert which HTTP headers the OpenAI client set. Tests should not break when an internal helper is renamed or a private function is extracted.

The v1 test suite is intentionally narrow: **unit tests only**, no integration tests, no Testcontainers, no Ktor `testApplication`. This trades coverage of the SQL and HTTP layers (which will be exercised manually during dev) for fast feedback and a small mental model. Integration tests are deferred to v1.5.

### Stack

- **JUnit 5** — test runner; standard on the JVM.
- **Kotest assertions** (`kotest-assertions-core`) for fluent, idiomatic Kotlin assertions and better failure messages. Kotest framework runner not used — JUnit 5 is enough.
- **MockK** — Kotlin-native mocking. Handles `suspend` functions cleanly.

### Modules tested in v1

- **`Config`** — parses well-formed env, rejects missing required values, applies defaults for optional values.
- **`EnrichmentResult` JSON deserialization** — accepts well-formed structured outputs, rejects malformed JSON, handles refusal-style responses gracefully.
- **Retry decision logic** — given (attempts, error), returns the correct outcome (retry-with-delay vs fail-terminal).
- **Status state machine** — `Status.next(event)` transitions are correct for every legal pair; illegal transitions throw or return an error type.
- **MIME and duration validators** — accept supported MIME types, reject unsupported; accept durations in the allowed window, reject outside.
- **`UserResolver`** — v1 implementation returns the hardcoded seeded user id.

### Explicitly out of scope for testing in v1

- `ThoughtRepository` SQL — needs real Postgres; defer to integration tests.
- `S3BlobStore` and the upload streaming path — needs MinIO or S3; defer.
- HTTP route handlers — would benefit from Ktor `testApplication`; defer.
- `ThoughtPipeline` orchestration — straddles unit/integration; the pieces it composes are unit-tested individually.
- Frontend tests — exercise manually for v1.

### Prior art

None in this repository — it is greenfield. Idiomatic references: the Ktor sample project's tests (for HTTP testing when v1.5 lands) and the Kotest documentation's pattern of one nested `describe`/`context` per behavior class.

## Out of Scope

The following are explicitly deferred beyond v1:

- **Authentication.** No login, no signup, no password hashing, no JWT. A single seeded user. The `users` table and `thoughts.user_id` foreign key are in place so auth can be added without restructuring.
- **Editing, deleting, or undoing thoughts.** No `PATCH /thoughts/:id`, no `DELETE`. Audio and transcripts are immutable in v1.
- **Manual transcript correction.** Whisper output is what the user sees.
- **Full-text search across transcripts.** Filtering is limited to category and tag for v1. Search comes later via Postgres full-text search or pgvector for semantic similarity.
- **Semantic clustering / "find similar thoughts."** Future, likely via pgvector embeddings stored alongside enrichments.
- **Tag normalization or merging.** Tags are accepted as-is from the LLM; the inevitable tag explosion (e.g., `kotlin` vs `Kotlin`) is a known v2 problem.
- **Rate limits and abuse protection.** Single-user assumption removes the need until auth lands.
- **Mobile apps (native iOS or Android).** PWA only.
- **Live transcription / streaming STT.** Audio is uploaded as a complete blob after recording stops.
- **Direct-to-S3 uploads via presigned URLs.** Audio streams through the backend to S3. Migration is cheap if needed.
- **Job queue infrastructure** (SQS, RabbitMQ, etc.). In-process coroutine worker is sufficient for v1.
- **Terraform / IaC.** Console-driven setup for v1. Port to Terraform once the architecture stabilizes.
- **APM / distributed tracing / Prometheus / Grafana.** Stdout logs + Sentry only.
- **CloudWatch Logs shipment.** Not needed — Cloud Run captures stdout natively.
- **Docker for local dev of the Kotlin app itself.** Docker Compose runs Postgres and MinIO only; the app runs natively for fast iteration.
- **End-to-end and integration tests.** Unit tests only for v1.

## Further Notes

- **Cost ceiling.** Whisper is the dominant cost driver (~$0.006/min). At 180 one-minute thoughts/month the audio bill is ~$1.10/month. `gpt-4o-mini` enrichment is negligible (fractions of a cent per thought). S3 storage at this volume is rounding error. Cloud Run, Vercel, Neon, GitHub Actions, and Sentry are all on free tiers that this project will not exceed.
- **Cold-start tradeoff.** Cloud Run and Neon both scale to zero. Combined first-request latency after idle is in the 5–8 second range. The PWA mitigates this with a `/healthz` keepalive ping on mount; users typically have a few seconds between opening the app and tapping record, which warms both services in parallel.
- **AWS in the stack.** S3 remains on AWS even though compute moved to GCP. The developer explicitly wants AWS exposure, and S3 is the canonical object-store API to know. Cross-cloud authentication uses an AWS IAM user with `s3:PutObject` / `s3:GetObject` scoped to the project's bucket(s).
- **Learning goals (non-functional but load-bearing).** The choice of Ktor over Spring, raw SQL via kotliquery over an ORM, in-process coroutines over an external queue, and a hand-rolled OpenAI client over the official Java SDK are all chosen to maximize exposure to idiomatic Kotlin and the JVM ecosystem.
- **Future evolution to multi-user.** When auth lands, the `UserResolver` swap is the only meaningful code change in the request path. The schema (`users` table, `user_id` FK on `thoughts`) already supports per-user isolation; the queries already filter by `user_id`.
- **Domain model invariant.** A `thoughts` row always has audio in S3 before transcription begins. A `thought_enrichments` row only exists once transcription has succeeded. The pipeline preserves these invariants; the schema does not enforce them (`thought_enrichments.thought_id` is just an FK, no partial-status constraint).
