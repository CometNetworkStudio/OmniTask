#!/bin/sh
set -eu

TEMPLATE="$1"
BIN="$2"
shift 2

CONFIG_DIR="${OMNITASK_CONFIG_DIR:-/config}"
CONFIG_FILE="$CONFIG_DIR/config.yaml"

mkdir -p "$CONFIG_DIR" /data/scripts /data/logs /data/workspaces /data/rate
if [ ! -f "$CONFIG_FILE" ]; then
  cp "$TEMPLATE" "$CONFIG_FILE"
fi

exec "$BIN" -conf "$CONFIG_FILE" "$@"
