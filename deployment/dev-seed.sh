#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
source backend/.env
: "${DATABASE_URL:?Set DATABASE_URL in backend/.env}"
command -v psql >/dev/null || {
    printf 'Install the PostgreSQL client (psql), then rerun make seed.\n' >&2
    exit 1
}

psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -f tests/db-mock-data/seed.sql
printf 'Demo spaces and resources are ready.\n'
