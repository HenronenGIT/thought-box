# 02 — DB + config + seeded user

**Type:** HITL
**Parent PRD:** `docs/prd/v1-thought-box.md`

## What to build

Stand up the Postgres dependency end-to-end and prove environment isolation. Neon hosts production Postgres; Docker Compose runs Postgres locally. Flyway migrations create a minimal `users` table with one seeded user. A `Config` module loads all runtime configuration from environment variables. A `UserResolver` returns the seeded user id (the single swap point for future auth). A `GET /me` endpoint returns the current user id. The backend logs the active environment (dev or prod) and the Postgres host it connected to at startup.

This slice exists so that every subsequent slice has a real DB, a real user id to attach data to, and an unambiguous way to read configuration.

## Acceptance criteria

- [ ] `Config` module loads typed configuration from env at startup; missing required values cause boot failure with a clear error.
- [ ] Flyway integrated; runs migrations on application startup.
- [ ] Initial migration creates `users` table (`id` uuid PK, `created_at`).
- [ ] Seed migration inserts exactly one user with a stable known UUID.
- [ ] kotliquery wired up with a connection pool reading from `Config`.
- [ ] `UserResolver` returns the seeded user id.
- [ ] `GET /me` returns `{ "user_id": "<uuid>" }`.
- [ ] Startup banner logs the environment name and Postgres host (no credentials in logs).
- [ ] `docker-compose.yml` at repo root runs Postgres for local dev; `.env.example` documents required env vars.
- [ ] Dev and prod use distinct Postgres databases (`thoughts_dev` vs `thoughts_prod`) on distinct Neon projects/branches.
- [ ] Unit tests: `Config` parsing (required-missing rejected, defaults applied) and `UserResolver` (returns expected id).

## Blocked by

- Blocked by #01
