#!/bin/sh
set -e

mkdir -p /data /app/tmp

# Bind-mounted repo is owned by host UID; git refuses otherwise (breaks go build -buildvcs).
git config --global --add safe.directory /app 2>/dev/null || true

exec "$@"
