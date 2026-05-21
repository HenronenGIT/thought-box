# Cloud Run Startup Failure — Troubleshooting Guide

The container builds and pushes successfully but Cloud Run rejects the revision with:
> "The user-provided container failed to start and listen on the port…"

This means the process crashes **before** binding to port 8080. Work through these steps in order.

---

## Step 1 — Read the Cloud Run logs

This is the fastest way to find the root cause.

1. Open the **Logs URL** printed at the bottom of the GitHub Actions failure output.
2. Look for lines with severity `ERROR` or `CRITICAL`.
3. The most likely messages:
   - `Missing required env var: <NAME>` → go to **Step 3**
   - A JDBC/Flyway exception → go to **Step 2**
   - Any other exception → note it and investigate from there.

---

## Step 2 — Verify the DATABASE_URL reaches Cloud Run intact

The URL contains special characters (`?`, `&`, `=`) that can be silently truncated when passed through `--set-env-vars`.

1. In the [GCP Console](https://console.cloud.google.com/run), open the **thoughtbox-backend** service.
2. Click **Edit & Deploy New Revision** → scroll to **Variables & Secrets**.
3. Find `DATABASE_URL` and confirm the full value is present:
   ```
   jdbc:postgresql://<host>/neondb?user=neondb_owner&password=<pw>&sslmode=require&channelBinding=require
   ```
   If it is truncated at the first `&` or `=`, the app will fail to connect.

4. While here, confirm every other required variable is present and non-empty:
   - `DATABASE_USER`
   - `DATABASE_PASSWORD`
   - `S3_BUCKET`, `S3_REGION`
   - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
   - `OPENAI_API_KEY`

---

## Step 3 — Verify GitHub Secrets are non-empty

A secret can exist with an empty value and still pass the workflow validation step.

1. Go to your GitHub repo → **Settings** → **Secrets and variables** → **Actions**.
2. Check that each secret below exists **and has a non-empty value** (GitHub shows `Updated X days ago` — if it was never set it shows nothing):

   | Secret | Notes |
   |---|---|
   | `DATABASE_URL` | Full JDBC URL including query string |
   | `DATABASE_USER` | Plain username, no quotes |
   | `DATABASE_PASSWORD` | Plain password, no quotes |
   | `S3_BUCKET` | Bucket name only |
   | `S3_REGION` | e.g. `eu-north-1` |
   | `AWS_ACCESS_KEY_ID` | |
   | `AWS_SECRET_ACCESS_KEY` | |
   | `OPENAI_API_KEY` | |
   | `CORS_ALLOWED_ORIGINS` | Optional, but set it to your frontend URL |

3. If any are missing or empty, update them and re-run the workflow.

---

## Step 4 — Check `channelBinding=require` compatibility

Your `DATABASE_URL` includes `channelBinding=require`. This requires **pgjdbc ≥ 42.7.0**. If your `build.gradle` pins an older driver version, Flyway will throw during the migration step on startup.

1. Open `build.gradle.kts` (or `build.gradle`) and find the PostgreSQL driver dependency, e.g.:
   ```
   implementation("org.postgresql:postgresql:<version>")
   ```
2. Confirm the version is **42.7.0 or higher**.
3. If it is older, either upgrade the driver or remove `channelBinding=require` from `DATABASE_URL`.

---

## Step 5 — Test the container locally against production secrets

Run the exact image Cloud Run uses, with the same env vars, to reproduce the crash locally.

```bash
# Pull the image that was just pushed (use the SHA from the Actions log)
IMAGE="<GCP_REGION>-docker.pkg.dev/<GCP_PROJECT_ID>/<ARTIFACT_REPOSITORY>/thought-box:<SHA>"
docker pull "$IMAGE"

# Run it with production env vars
docker run --rm \
  -e PORT=8080 \
  -e APP_ENV=prod \
  -e DATABASE_URL="<your DATABASE_URL>" \
  -e DATABASE_USER="<your DATABASE_USER>" \
  -e DATABASE_PASSWORD="<your DATABASE_PASSWORD>" \
  -e CORS_ALLOWED_ORIGINS="<your CORS_ALLOWED_ORIGINS>" \
  -e S3_BUCKET="<your S3_BUCKET>" \
  -e S3_REGION="<your S3_REGION>" \
  -e AWS_ACCESS_KEY_ID="<your AWS_ACCESS_KEY_ID>" \
  -e AWS_SECRET_ACCESS_KEY="<your AWS_SECRET_ACCESS_KEY>" \
  -e OPENAI_API_KEY="<your OPENAI_API_KEY>" \
  -p 8080:8080 \
  "$IMAGE"
```

Watch the output. Any startup exception will be printed directly to the terminal before the server starts.

---

## Step 6 — Verify Neon DB is reachable from Cloud Run's region

Cloud Run runs in `europe-north2` (GCP). The Neon pooler is in `eu-central-1` (AWS). Cross-cloud connections are normally fine, but Neon projects can have IP allowlists.

1. Log in to [Neon Console](https://console.neon.tech) → your project → **Settings** → **IP Allow**.
2. If IP allowlisting is enabled, either disable it or add Cloud Run's outbound IPs (these are not static by default — disable the allowlist instead).

---

## Most likely culprits (ranked)

| # | Cause | Where to confirm |
|---|---|---|
| 1 | `DATABASE_URL` truncated in Cloud Run env vars | Step 2 |
| 2 | A required GitHub Secret is empty | Step 3 |
| 3 | Flyway crash due to DB unreachable | Step 1 logs + Step 6 |
| 4 | `channelBinding=require` + old pgjdbc driver | Step 4 |
