# Setup Checklist

## GCP / Cloud Run

- Create GCP project.
- Enable Artifact Registry and Cloud Run APIs.
- Create Docker Artifact Registry repo.
- Create deploy service account with Artifact Registry writer + Cloud Run admin.
- Add GitHub secrets: `GCP_PROJECT_ID`, `GCP_REGION`, `ARTIFACT_REPOSITORY`, `CLOUD_RUN_SERVICE`, `GCP_SERVICE_ACCOUNT_JSON`.
- Set Cloud Run env vars from `.env.example`, using prod values.

## Neon

- Create separate dev/prod DBs or branches: `thoughts_dev`, `thoughts_prod`.
- Put prod JDBC URL/user/password in Cloud Run env.
- Keep local Docker Postgres for dev.

## AWS / S3

- Create `thoughts-dev` and `thoughts-prod` buckets.
- Create IAM user with `s3:PutObject`, `s3:GetObject`, `s3:HeadObject` scoped to bucket objects.
- Store access key + secret in local env and Cloud Run env.

## OpenAI

- Create separate dev/prod API keys.
- Set `OPENAI_API_KEY` per environment.

## Google OAuth (sign-in)

- In the GCP console: APIs & Services → Credentials → Create Credentials → OAuth client ID.
- Application type: **Web application**.
- Authorized redirect URIs:
  - Dev: `http://localhost:8080/auth/google/callback`
  - Prod: `https://<your-api-host>/auth/google/callback`
- Copy the generated client ID + secret into `GOOGLE_OAUTH_CLIENT_ID` and `GOOGLE_OAUTH_CLIENT_SECRET`.
- Set `GOOGLE_OAUTH_REDIRECT_URL` to the exact URI registered above for that environment.
- Set `WEB_BASE_URL` to where users should land after sign-in (`http://localhost:3000` in dev).
- Set `SESSION_SIGNING_KEY` to a base64-encoded 32 random bytes. Generate with:
  ```
  openssl rand -base64 32
  ```
  Use a different key per environment. Treat it as a secret — rotating it logs all users out.

## Local run

- Make sure Docker Desktop is running.
- Start dependencies: `docker compose up -d postgres minio`
- Create the MinIO bucket: `docker exec kotlin-minio-1 mc mb local/thoughts-dev` (one-time, persists in the `minio-data` volume)
- Run frontend: `npm --prefix apps/web run dev`
- Run backend: `./gradlew runDev`
- Verify: `curl http://localhost:8080/healthz`
- Stop backend with `Ctrl+C`.
- Gradle wrapper auto-downloads Gradle and a JDK 21 toolchain on first run.

## Vercel

- Import `apps/web`.
- Set `NEXT_PUBLIC_API_BASE_URL` to Cloud Run URL.
- Deploy.
