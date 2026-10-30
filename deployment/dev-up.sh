#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

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

wait -n "$backend_pid" "$frontend_pid"
