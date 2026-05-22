#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [ -f "$ROOT_DIR/.env" ]; then
  set -a
  . "$ROOT_DIR/.env"
  set +a
fi

if [ -f "$ROOT_DIR/apps/api-go/.env" ]; then
  set -a
  . "$ROOT_DIR/apps/api-go/.env"
  set +a
fi

: "${APP_ENV:=dev}"
: "${PORT:=8080}"
: "${DATABASE_USER:=thoughts}"
: "${DATABASE_PASSWORD:=thoughts}"
: "${CORS_ALLOWED_ORIGINS:=http://localhost:3000}"
: "${S3_BUCKET:=thoughts}"
: "${S3_REGION:=us-east-1}"
: "${S3_ENDPOINT:=http://localhost:9000}"
: "${AWS_ACCESS_KEY_ID:=minioadmin}"
: "${AWS_SECRET_ACCESS_KEY:=minioadmin}"
: "${MAX_THOUGHT_DURATION_MS:=60000}"
: "${MIN_THOUGHT_DURATION_MS:=1000}"
: "${MAX_THOUGHT_SIZE_BYTES:=10485760}"

case "${DATABASE_URL:-}" in
  postgres://*|postgresql://*) ;;
  *)
    DATABASE_URL="postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@localhost:5432/thoughts_dev?sslmode=disable"
    ;;
esac

export APP_ENV PORT DATABASE_URL CORS_ALLOWED_ORIGINS
export S3_BUCKET S3_REGION S3_ENDPOINT AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY
export OPENAI_API_KEY MAX_THOUGHT_DURATION_MS MIN_THOUGHT_DURATION_MS MAX_THOUGHT_SIZE_BYTES

cd "$ROOT_DIR/apps/api-go"
exec go run ./cmd/api
