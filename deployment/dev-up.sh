#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

for command in curl psql; do
    command -v "$command" >/dev/null || {
        printf 'Install %s before running make dev_up.\n' "$command" >&2
        exit 1
    }
done

"${MAKE:-make}" db_up frontend_deps
(
    cd backend
    go build -o main cmd/main.go
)

backend_pid=''
frontend_pid=''
cleanup() {
    status=$?
    trap - EXIT INT TERM
    for pid in "$backend_pid" "$frontend_pid"; do
        if [[ -n "$pid" ]]; then
            kill "$pid" 2>/dev/null || true
        fi
    done
    for pid in "$backend_pid" "$frontend_pid"; do
        if [[ -n "$pid" ]]; then
            wait "$pid" 2>/dev/null || true
        fi
    done
    exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

(
    set -a
    source backend/.env
    set +a
    cd backend
    exec ./main
) &
backend_pid=$!

(
    cd frontend
    exec node node_modules/vite/bin/vite.js --strictPort
) &
frontend_pid=$!

ready=false
for attempt in {1..60}; do
    for pid in "$backend_pid" "$frontend_pid"; do
        if ! kill -0 "$pid" 2>/dev/null; then
            wait "$pid"
            exit 1
        fi
    done
    if curl --fail --silent --max-time 1 http://127.0.0.1:3000/v1/ready >/dev/null; then
        ready=true
        break
    fi
    sleep 0.5
done
if [[ "$ready" != true ]]; then
    printf 'Backend did not become ready for development seeding.\n' >&2
    exit 1
fi

"${MAKE:-make}" seed
wait -n "$backend_pid" "$frontend_pid"
