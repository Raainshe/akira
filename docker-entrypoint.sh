#!/bin/sh
set -e

mkdir -p /data
chown -R akira:akira /data

exec su-exec akira "$@"
