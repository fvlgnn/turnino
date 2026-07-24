#!/bin/sh
set -eu

counter_id="${TURNINO_COUNTER_ID:-}"
base_url="${TURNINO_BASE_URL:-http://localhost:8080}"

case "$counter_id" in
    ""|*[!0-9]*|0)
        echo "TURNINO_COUNTER_ID must be a positive integer." >&2
        exit 1
        ;;
esac

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
TURNINO_MONITOR_URL="${base_url%/}/counter-display?counter=${counter_id}"
export TURNINO_MONITOR_URL
exec "$script_dir/start-monitor.sh"
