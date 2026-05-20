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

## Vercel

- Import `apps/web`.
- Set `NEXT_PUBLIC_API_BASE_URL` to Cloud Run URL.
- Deploy.

