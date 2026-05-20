# 03 — Audio upload + playback (no transcription)

**Type:** HITL
**Parent PRD:** `docs/prd/v1-thought-box.md`

## What to build

End-to-end audio capture and playback, without any transcription or enrichment. The PWA records audio via `MediaRecorder`, picks a supported MIME type via per-browser negotiation, enforces the duration cap fetched from `/config`, and uploads the blob to the backend as multipart. The backend streams the blob to S3 (production) or MinIO (local dev) via the `BlobStore` abstraction, inserts a `thoughts` row, and returns its id. The PWA can fetch the thought back and replay the original audio.

This slice proves the file pipeline (browser → backend → object storage → backend → browser) without involving any external AI services. By the end, the user can record a one-minute clip on the deployed PWA and play it back.

## Acceptance criteria

- [ ] `BlobStore` interface defined (put, get, exists).
- [ ] `S3BlobStore` implementation wraps the AWS SDK; reads endpoint, region, bucket, credentials from `Config`.
- [ ] In dev, `S3_ENDPOINT` env var points `S3BlobStore` at MinIO (no `if isDev` branches in code).
- [ ] `docker-compose.yml` updated to include MinIO alongside Postgres.
- [ ] Flyway migration creates `thoughts` table with: `id`, `user_id` (FK), `audio_s3_key`, `mime_type`, `duration_ms`, `size_bytes`, `status` (default `uploaded` for this slice), `created_at`, `updated_at`.
- [ ] `GET /config` returns `{ max_duration_ms, min_duration_ms, max_size_bytes }` from `Config`.
- [ ] `POST /thoughts` accepts multipart (audio blob + `mime_type` + `duration_ms` form fields), streams the audio directly to `BlobStore` (no full-buffer in memory), inserts a row, returns `{ id, status, created_at }`.
- [ ] Server-side validation: rejects durations outside `[MIN, MAX]` and sizes over `MAX_THOUGHT_SIZE_BYTES` with HTTP 400.
- [ ] `GET /thoughts/:id` returns the row's metadata (status + audio info; transcript/enrichment fields null).
- [ ] `GET /thoughts/:id/audio` streams the original audio with the stored `Content-Type`.
- [ ] PWA recorder component: requests mic permission, negotiates MIME (`audio/webm;codecs=opus` → `audio/mp4` → fallback), enforces duration cap with auto-stop, shows elapsed-time display.
- [ ] PWA fetches `/config` on mount and uses returned limits to drive client-side validation.
- [ ] PWA uploads recorded blob to `POST /thoughts`, then fetches `GET /thoughts/:id/audio` and plays it back.
- [ ] Cross-cloud auth: Cloud Run service has env vars holding AWS IAM access key + secret scoped to `s3:PutObject`/`s3:GetObject` on the production bucket.
- [ ] Unit tests: MIME validator (accept supported, reject unsupported); duration validator (accept in-window, reject out-of-window).

## Blocked by

- Blocked by #02
