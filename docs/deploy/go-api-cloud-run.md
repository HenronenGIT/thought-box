# Go API Cloud Run

Separate service first:

```sh
gcloud run deploy thought-box-api-go \
  --source apps/api-go \
  --region "$GCP_REGION" \
  --allow-unauthenticated \
  --set-env-vars APP_ENV=prod,CORS_ALLOWED_ORIGINS="$WEB_ORIGIN" \
  --set-env-vars DATABASE_URL="$DATABASE_URL" \
  --set-env-vars S3_BUCKET="$S3_BUCKET",S3_REGION="$S3_REGION" \
  --set-env-vars WORKER_ENABLED=true \
  --set-secrets OPENAI_API_KEY=OPENAI_API_KEY:latest,AWS_ACCESS_KEY_ID=AWS_ACCESS_KEY_ID:latest,AWS_SECRET_ACCESS_KEY=AWS_SECRET_ACCESS_KEY:latest
```

Keep Kotlin service live. Point a staging web build at `thought-box-api-go` with `NEXT_PUBLIC_API_BASE_URL`.

Required env:

- `DATABASE_URL`: `postgres://user:password@host:port/db`, not JDBC
- `OPENAI_API_KEY`
- `S3_BUCKET`
- `S3_REGION`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `CORS_ALLOWED_ORIGINS`

Cloud Run caveat:

- Do not use `/healthz` externally. Cloud Run reserves some URL paths ending in `z`; use `/health`.

Smoke checks:

```sh
curl "$GO_API_URL/health"
curl "$GO_API_URL/config"
curl "$GO_API_URL/me"
curl "$GO_API_URL/thoughts"
```
