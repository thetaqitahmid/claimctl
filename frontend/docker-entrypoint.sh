#!/bin/sh
set -eu
export TRUSTED_INGRESS_CIDR="${TRUSTED_INGRESS_CIDR:-127.0.0.1/32}"
envsubst '$BACKEND_HOST $BACKEND_PORT $TRUSTED_INGRESS_CIDR' < /etc/nginx/conf.d/default.conf.template > /etc/nginx/conf.d/default.conf

nginx -t
exec nginx -g 'daemon off;'