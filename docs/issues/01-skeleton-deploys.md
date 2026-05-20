# 01 — End-to-end skeleton deploys

**Type:** HITL
**Parent PRD:** `docs/prd/v1-thought-box.md`

## What to build

A minimal but real deployment of both halves of the stack, end-to-end. A Kotlin/Ktor backend serves a single `GET /healthz` endpoint, runs in a Docker container on Google Cloud Run, and is reachable at its `*.run.app` HTTPS URL. A Next.js (App Router) frontend deployed to Vercel pings `/healthz` on mount and renders the response. CI runs on every push; a manual deploy workflow ships the backend.

The goal is to prove the integration: code in two repos (or one monorepo with two apps), two deploy targets, two CI flows, and one HTTPS call from frontend to backend. No database, no S3, no OpenAI. Everything subsequent stacks on top of this.

## Acceptance criteria

- [ ] Backend project scaffolded with Gradle Kotlin DSL, version catalog, JDK 21, Ktor.
- [ ] Multi-stage Dockerfile builds a fat JAR (via Shadow plugin) and runs it on `eclipse-temurin:21-jre-alpine`.
- [ ] `GET /healthz` returns `200 OK` with a JSON body.
- [ ] GitHub Actions CI runs `./gradlew test build` on every push and PR; green on `main`.
- [ ] GitHub Actions `workflow_dispatch` job builds the image, pushes to Google Artifact Registry, and runs `gcloud run deploy`.
- [ ] Cloud Run service is reachable at its `*.run.app` URL over HTTPS.
- [ ] Next.js (App Router) project scaffolded and deployed to Vercel.
- [ ] On page load, the PWA fetches `/healthz` from the Cloud Run URL and renders the response.
- [ ] CORS configured on the backend to allow the Vercel domain.
- [ ] Cold-start mitigation: the keepalive `/healthz` ping is intentionally the same call as the page-load probe, so it both proves connectivity and warms the service.
- [ ] One-time setup steps documented (GCP project, Artifact Registry repo, Cloud Run service, service account + roles, GitHub secrets, Vercel project).

## Blocked by

None — can start immediately.
