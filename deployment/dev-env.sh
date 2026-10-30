#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
config="$root/backend/.env"
if [[ -e "$config" ]]; then
    exit 0
fi

command -v openssl >/dev/null || {
    printf 'OpenSSL is required to generate local development credentials.\n' >&2
    exit 1
}

umask 077
temporary=$(mktemp "$root/backend/.env.XXXXXX")
trap 'rm -f "$temporary"' EXIT
password="Dev1!$(openssl rand -hex 20)"
encryption_key=$(openssl rand -base64 32)
awk -v password="$password" -v encryption_key="$encryption_key" '
    /^ADMIN_PASSWORD=/ { print "ADMIN_PASSWORD=\"" password "\""; next }
    /^APP_ENCRYPTION_KEY=/ { print "APP_ENCRYPTION_KEY=\"" encryption_key "\""; next }
    /^OIDC_[A-Z_]+=/ { print "# " $0; next }
    { print }
' "$root/backend/.env.example" > "$temporary"
if ln "$temporary" "$config" 2>/dev/null; then
    printf 'Created backend/.env with private local development credentials.\n'
    printf 'Sign in as admin@example.com using ADMIN_PASSWORD from that file.\n'
fi
