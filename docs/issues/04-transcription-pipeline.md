# 04 — Transcription pipeline

**Type:** HITL
**Parent PRD:** `docs/prd/v1-thought-box.md`

## What to build

Add the asynchronous transcription stage end-to-end. A `Transcriber` interface with an `OpenAiTranscriber` implementation calls Whisper. A `ThoughtPipeline` runs as a coroutine worker on a dedicated `CoroutineScope` owned by the application; it picks `pending` rows, transitions them through `transcribing` to `done`, and handles retries and failures. A `StartupRecovery` sweep on boot reclaims rows stuck in `transcribing` by flipping them back to `pending`. The PWA polls `GET /thoughts/:id` and renders the transcript when it appears.

After this slice, the user can record a thought, see it transcribed automatically within seconds, and recover gracefully from a transient Whisper failure or a backend restart.

## Acceptance criteria

- [ ] `Transcriber` interface defined: `suspend fun transcribe(audio: AudioBlob): TranscriptionResult`.
- [ ] `OpenAiTranscriber` implementation calls the Whisper endpoint using Ktor `HttpClient` + kotlinx.serialization (no official OpenAI SDK).
- [ ] OpenAI API key loaded from `Config`; dev and prod use distinct keys.
- [ ] Flyway migration adds `transcript` (text, nullable), `status` (text, replaces default-`uploaded` with proper state machine), `attempts` (int, default 0), `last_error` (text, nullable), `last_attempt_at` (timestamptz, nullable), `transcribed_at` (timestamptz, nullable) to `thoughts`.
- [ ] Status state machine implemented as a pure function: `Status.next(event)` covers transitions `uploaded → pending → transcribing → done`, plus `failed_transcription` terminal.
- [ ] Retry decision logic implemented as a pure function: given (`attempts`, error category), returns `RetryAfter(duration)` or `Fail`. Three attempts, exponential backoff (1s/4s/16s).
- [ ] `ThoughtPipeline` runs as a coroutine worker owned by a single application-scoped `CoroutineScope`; polls for `pending` rows and drives them through the state machine.
- [ ] On Whisper failure, the worker increments `attempts`, sets `last_error`, sleeps the backoff, retries. After 3 failed attempts, status flips to `failed_transcription`.
- [ ] On Whisper success, transcript is stored, status flips to `done`, `transcribed_at` is populated.
- [ ] `StartupRecovery` sweep: on boot, any row in `transcribing` is reset to `pending`.
- [ ] `GET /thoughts/:id` now returns `transcript` when present and current `status`.
- [ ] PWA polls `GET /thoughts/:id` at 2-second intervals while status ∈ {`pending`, `transcribing`}; renders transcript when status is `done`; renders a clear failure state when status is `failed_transcription`.
- [ ] Unit tests: status state machine (every legal transition; illegal transitions rejected); retry decision logic (each attempt count → expected outcome); `OpenAiTranscriber` happy-path and error-path with MockK fakes; `Transcriber` fake implementation used in tests.

## Blocked by

- Blocked by #03
