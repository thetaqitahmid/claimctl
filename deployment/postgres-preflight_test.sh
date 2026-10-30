#!/bin/sh
set -eu

base=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixtures=$(mktemp -d /tmp/claimctl-postgres-layout.XXXXXX)
trap 'rm -rf "$fixtures"' EXIT HUP INT TERM

check() {
    name=$1
    expected=$2
    pgdata=${3:-/var/lib/postgresql/18/docker}
    mkdir -p "$fixtures/$name"
    if docker run --rm --network none \
        --entrypoint /usr/local/bin/claimctl-postgres-preflight \
        -e "PGDATA=$pgdata" \
        -v "$base/deployment/postgres-preflight.sh:/usr/local/bin/claimctl-postgres-preflight:ro" \
        -v "$fixtures/$name:/var/lib/postgresql:ro" \
        postgres:18.1-alpine true > "$fixtures/output" 2>&1; then
        actual=pass
    else
        actual=fail
    fi
    [ "$actual" = "$expected" ] || {
        echo "FAIL: $name expected $expected, got $actual" >&2
        exit 1
    }
    echo "PASS: $name ($actual)"
}

check fresh pass
mkdir -p "$fixtures/current/18/docker"
printf '18\n' > "$fixtures/current/18/docker/PG_VERSION"
check current pass
mkdir -p "$fixtures/root-legacy"
printf '18\n' > "$fixtures/root-legacy/PG_VERSION"
check root-legacy fail
mkdir -p "$fixtures/legacy-data/data"
printf '18\n' > "$fixtures/legacy-data/data/PG_VERSION"
check legacy-data fail
mkdir -p "$fixtures/old-version/18/docker"
printf '17\n' > "$fixtures/old-version/18/docker/PG_VERSION"
check old-version fail
mkdir -p "$fixtures/no-version/18/docker/base"
check no-version fail
mkdir -p "$fixtures/unexpected-root/17/docker"
check unexpected-root fail
mkdir -p "$fixtures/unexpected-version/18/other"
check unexpected-version fail
mkdir -p "$fixtures/symlink"
ln -s /tmp "$fixtures/symlink/18"
check symlink fail
check wrong-pgdata fail /var/lib/postgresql/data
