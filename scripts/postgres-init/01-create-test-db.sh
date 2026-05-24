#!/bin/bash
set -e

# Create a dedicated test database. Only runs on first container start
# (when the postgres data volume is empty). Owned by the same role as the
# dev DB so app code reuses the existing connection credentials.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE thoughts_test OWNER ${POSTGRES_USER};
EOSQL
