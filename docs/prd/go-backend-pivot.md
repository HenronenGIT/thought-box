# PRD — Go Backend Pivot

## Problem Statement

The current backend is implemented in Kotlin, but the developer's learning goal has shifted to Go. The backend also needs a lower-friction production deployment path. The project should pivot toward a Go backend without rewriting the product, changing the frontend contract unnecessarily, or losing the existing backend behavior that already works.

## Solution

Build a new Go backend in the monorepo as the future production backend. The Kotlin backend remains as a reference implementation until the Go backend reaches parity. The Go backend exposes the same HTTP API contract, uses the same core product model, stores audio in S3-compatible storage, uses Postgres as the source of truth, transcribes and enriches thoughts through OpenAI, and runs a background worker in-process.

The new backend lives in a separate application folder and deploys through a separate Dockerfile and separate Cloud Run service first. Production cutover happens only after parity is verified.

## User Stories

1. As the developer, I want the backend written in Go, so that I can learn Go through a real production application.
2. As the developer, I want the Go backend isolated in its own application folder, so that the repository can become a monorepo cleanly.
3. As the developer, I want the Kotlin backend to remain available as a reference, so that I can compare behavior while porting.
4. As the developer, I want the web app to keep using the same backend API, so that frontend work does not block the backend pivot.
5. As the developer, I want a separate Go Dockerfile, so that the Go backend can be built and deployed independently.
6. As the developer, I want the Go backend deployed to a separate Cloud Run service first, so that I can test it without breaking the current backend.
7. As the developer, I want Go tool versions and common commands managed through mise, so that local development stays repeatable.
8. As the developer, I want local dependencies to run in Docker Compose, so that Postgres and MinIO are easy to start.
9. As the developer, I want the Go app to run on the host during development, so that edit-run feedback is fast.
10. As the developer, I want the Go backend to use chi, so that routing and middleware stay small and idiomatic.
11. As the developer, I want database access through raw pgx, so that SQL remains explicit and I learn Go/Postgres behavior directly.
12. As the developer, I want migrations managed by goose, so that schema changes are versioned with common Go tooling.
13. As the developer, I want migrations to run on startup, so that local and deployed databases stay aligned with the app version.
14. As the developer, I want OpenAI calls made through the OpenAI Go SDK, so that provider integration uses maintained client code.
15. As the developer, I want S3 access through AWS SDK for Go v2, so that production S3 and local MinIO can share the same abstraction.
16. As the developer, I want the Go backend to accept a pgx-native DATABASE_URL, so that database configuration matches Go conventions.
17. As the developer, I want DATABASE_URL to contain username and password, so that separate DATABASE_USER and DATABASE_PASSWORD env vars are unnecessary.
18. As the developer, I want the Go backend to fail fast when DATABASE_URL is missing or invalid, so that configuration mistakes are obvious.
19. As the developer, I want the old JDBC URL separated as JDBC_DATABASE_URL if needed for Kotlin, so that Go config is not ambiguous.
20. As a user, I want to record and upload audio thoughts the same way as before, so that the backend pivot does not change the product flow.
21. As a user, I want uploaded audio preserved in S3-compatible storage, so that I can replay the original thought later.
22. As a user, I want uploaded thoughts inserted as pending records, so that processing can continue asynchronously.
23. As a user, I want thoughts transcribed automatically, so that my spoken notes become readable text.
24. As a user, I want thoughts enriched automatically, so that each thought has a title, summary, category, and tags.
25. As a user, I want thought status to reflect processing progress, so that I can understand whether a thought is pending, processing, complete, or failed.
26. As a user, I want failed processing retried, so that transient OpenAI or storage errors do not permanently lose work.
27. As a user, I want failed terminal states recorded, so that problems are visible rather than silent.
28. As a user, I want a backend restart not to strand work forever, so that interrupted processing can recover on startup.
29. As a user, I want thoughts listed newest-first, so that recent thoughts are easy to find.
30. As a user, I want cursor pagination, so that thought lists remain fast as data grows.
31. As a user, I want category filtering, so that I can browse a specific kind of thought.
32. As a user, I want tag filtering, so that I can find related thoughts by topic.
33. As a user, I want only the categories idea, observation, feeling, and learning, so that categorization stays focused.
34. As a user, I want category values stable in the API, so that filters and UI labels behave consistently.
35. As a developer, I want Go constants internally with lowercase wire values externally, so that code is idiomatic while DB/API remain simple.
36. As a developer, I want request correlation ids accepted or generated, so that logs can be tied to individual requests.
37. As a developer, I want structured logs through slog, so that production behavior can be inspected in Cloud Run.
38. As a developer, I want custom CORS middleware, so that behavior matches the existing backend without extra dependency behavior.
39. As a developer, I want Sentry deferred, so that the first port focuses on Go parity rather than observability expansion.
40. As a developer, I want tests added after feature parity, so that the first phase optimizes learning and porting speed.
41. As a developer, I want tests for config, validation, repository, handlers, and worker transitions, so that the eventual production replacement has coverage around important behavior.

## Implementation Decisions

- The repository becomes a monorepo with the Go backend in a separate app folder and the existing web app kept in its current app folder.
- The Go backend is the intended production replacement. The Kotlin backend is retained as a reference until parity and cutover.
- The Go application uses a `cmd` plus `internal` layout.
- The HTTP package is named `httpapi` to avoid confusion with Go's standard `net/http` package.
- The router is chi.
- JSON handling uses the standard `encoding/json` package with explicit DTO structs.
- Unknown JSON fields are not globally rejected in the first version, to preserve compatibility and avoid surprise client breakage.
- The database driver is raw pgx.
- Migrations use goose and run on application startup.
- The Go app requires a pgx-native `DATABASE_URL` containing username and password.
- The Go app does not fallback to JDBC URLs. Missing or invalid `DATABASE_URL` is a startup error.
- `JDBC_DATABASE_URL` may remain for the Kotlin backend/reference, but Go does not depend on it.
- S3-compatible storage uses AWS SDK for Go v2.
- The storage layer supports an optional custom endpoint for MinIO.
- OpenAI integration uses the OpenAI Go SDK.
- The first Go version keeps the OpenAI enrichment behavior close to the Kotlin reference.
- Prompt/category behavior changes only for the new category enum.
- Valid categories are exactly `idea`, `observation`, `feeling`, and `learning`.
- Category values are lowercase in the DB and API.
- Go code may use typed constants internally for category values.
- The Go backend keeps the seeded user resolver behavior. Real auth is deferred.
- The Go backend exposes the same frontend-facing routes as the Kotlin backend.
- Error compatibility requires the same status codes and `{"error": string}` shape, but not exact message text.
- Multipart audio upload streams to a temp file before S3 upload to avoid buffering whole audio files in memory.
- The background worker remains in-process and polls Postgres.
- Row claiming uses the same Postgres pattern: `FOR UPDATE SKIP LOCKED`.
- Startup recovery returns interrupted work to retryable states.
- Logs use Go's `log/slog`.
- Requests accept or generate `X-Correlation-Id`, include it in logs, and return it in response headers.
- CORS is implemented through a small custom middleware matching current allowed headers and methods.
- Sentry is deferred.
- Local dev uses Docker Compose for Postgres and MinIO.
- Local app commands and tool versions are managed by root mise configuration.
- The Go backend has its own Dockerfile.
- The Go backend deploys first to a separate Cloud Run service before production cutover.

## Testing Decisions

- Tests are added after implementation reaches full parity.
- Good tests should verify external behavior and contracts, not private helper details.
- Config tests should verify required env validation, defaults, and fail-fast behavior for missing `DATABASE_URL`.
- Validation tests should cover MIME type, duration, size, pagination limit, cursor parsing, and category validation.
- Repository tests should exercise Postgres behavior around insert, list, filters, row claiming, status transitions, enrichment insert, and startup recovery.
- Handler tests should cover route status codes, JSON response shapes, multipart validation failures, not-found behavior, and user scoping.
- Worker tests should use fake transcriber/enricher/storage dependencies to verify status transitions, retries, terminal failures, and recovery behavior.
- OpenAI and S3 should be behind interfaces so unit tests do not require real network calls.
- Integration tests against local Postgres/MinIO are desirable after core parity.
- Existing Kotlin tests provide prior-art coverage areas: config parsing, validation, enrichment parsing, status behavior, user resolver behavior, and retry policy.

## Out of Scope

- Real authentication.
- Multi-user auth enforcement beyond the existing seeded user behavior.
- Rewriting the frontend beyond category/filter contract changes needed for the new enum.
- Queue infrastructure such as Cloud Tasks, Pub/Sub, SQS, or Redis.
- Sentry integration in the first Go port.
- Exact backend error message parity.
- ORM adoption.
- SQL code generation with sqlc.
- Declarative schema management with Atlas.
- Supporting legacy JDBC database URLs in the Go app.
- Deleting the Kotlin backend before Go parity and production cutover.

## Further Notes

The pivot is primarily a learning and production replacement effort, not a product redesign. The highest-value constraint is keeping the frontend-facing contract stable while swapping backend runtime and implementation language.

The category enum intentionally changes because there is no existing production data to preserve. The new canonical category set is `idea`, `observation`, `feeling`, and `learning`.

The migration path should favor simple, boring Go: chi for routing, raw pgx for SQL, goose for migrations, AWS SDK v2 for storage, OpenAI SDK for AI calls, and slog for logs.
