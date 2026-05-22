# Go Backend Parity

Regression checklist:

- `GET /health` returns `{ "ok": true, "env": string }`
- `GET /config` returns upload limits
- `GET /me` returns seeded user id
- `POST /thoughts` accepts multipart audio, stores S3 object, returns `201`
- Worker moves `pending -> transcribing -> enriching -> done`
- OpenAI transcription writes `transcript`
- OpenAI enrichment writes category/tags/title/summary
- `GET /thoughts` lists newest first
- `GET /thoughts?category=idea` filters
- `GET /thoughts?tag=x` filters
- `GET /thoughts/{id}` returns one thought
- `GET /thoughts/{id}/audio` streams audio bytes
- Bad IDs preserve `{"error": string}` shape
- Missing rows return `404`

Cutover gates:

- Staging web passes full record/list/playback flow against Go service
- Cloud Run logs show no repeated startup migration failures
- Worker terminal failures are visible via `last_error`
- Kotlin service remains available for rollback
- Web `NEXT_PUBLIC_API_BASE_URL` can switch back without schema change
