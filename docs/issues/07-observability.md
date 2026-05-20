# 07 — Observability (structured logs + Sentry)

**Type:** AFK
**Parent PRD:** `docs/prd/v1-thought-box.md`

## What to build

Make the running system inspectable. Logback with `logstash-logback-encoder` produces one JSON object per log line to stdout. Every HTTP request, every pipeline state transition, and every external API call emits a structured event with stable field names. Sentry's JVM SDK is wired into Ktor's exception handler so unhandled errors are reported with stack traces and grouped automatically. Sensitive payloads (transcript content, audio bytes, API keys) are never logged.

After this slice, debugging a stuck thought or a Whisper hiccup is a matter of filtering Cloud Run's log console by `thought_id` or `event`, and the developer gets an email when something genuinely explodes.

## Acceptance criteria

- [ ] Logback configured with `logstash-logback-encoder`; default appender emits JSON to stdout; one line per event.
- [ ] Per-request correlation id (UUID) propagated via MDC and included in every log line emitted during the request.
- [ ] Request logging: method, path, status, duration_ms, correlation id (no request bodies, no response bodies).
- [ ] Pipeline transition logging: `thought_id`, `from_status`, `to_status`, `attempts`, optional `error_class`.
- [ ] External API call logging: provider (`openai-whisper` / `openai-chat` / `s3`), duration_ms, response_status, optional `error_class`. No prompt content, no transcript, no audio bytes.
- [ ] Sentry SDK initialized at startup using DSN from `Config`; dev and prod use distinct DSNs (or environment tagging).
- [ ] Ktor exception handler reports unhandled exceptions to Sentry with the correlation id attached as a tag.
- [ ] Pipeline worker reports terminal failures (`failed_transcription`, `failed_enrichment`) to Sentry as warnings, with `thought_id` and `last_error` attached.
- [ ] PII guard: an automated check (e.g., a log-line redaction filter or a code review checklist documented in the issue) ensures transcript and API-key values never appear in logs.
- [ ] No new unit tests required beyond verifying any pure-function PII redaction helpers if used.

## Blocked by

- Blocked by #04
