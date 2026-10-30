#!/bin/sh
set -eu

fail() {
    echo "PostgreSQL layout preflight failed: $1. Manual inspection/upgrade required." >&2
    exit 1
}

root=/var/lib/postgresql
expected="$root/18/docker"
[ "${PGDATA:-}" = "$expected" ] || fail "PGDATA must be $expected"
[ ! -e "$root/PG_VERSION" ] || fail "legacy cluster at volume root"

for entry in "$root"/* "$root"/.[!.]* "$root"/..?*; do
    [ -e "$entry" ] || [ -L "$entry" ] || continue
    [ "$entry" = "$root/18" ] || fail "unexpected populated volume layout"
    [ -d "$entry" ] && [ ! -L "$entry" ] || fail "invalid version directory"
done

if [ -d "$root/18" ]; then
    for entry in "$root/18"/* "$root/18"/.[!.]* "$root/18"/..?*; do
        [ -e "$entry" ] || [ -L "$entry" ] || continue
        [ "$entry" = "$expected" ] || fail "unexpected version directory layout"
        [ -d "$entry" ] && [ ! -L "$entry" ] || fail "invalid data directory"
    done
fi

if [ -d "$expected" ]; then
    populated=false
    for entry in "$expected"/* "$expected"/.[!.]* "$expected"/..?*; do
        [ -e "$entry" ] || [ -L "$entry" ] || continue
        populated=true
    done
    if [ "$populated" = true ]; then
        [ -f "$expected/PG_VERSION" ] && [ ! -L "$expected/PG_VERSION" ] || fail "populated data directory has no version"
        [ "$(tr -d '\r\n' < "$expected/PG_VERSION")" = 18 ] || fail "data version is not 18"
    fi
fi

exec /usr/local/bin/docker-entrypoint.sh "$@"
