#!/bin/sh
set -e
OUT_LOG_PATH=${1:-/dev/null}
ERR_LOG_PATH=${2:-$OUT_LOG_PATH}
PID_FILE=${3}
shift 3
USER_COMMAND="$@"

echo "Starting service..."
nohup sh -c "$USER_COMMAND" > "$OUT_LOG_PATH" 2> "$ERR_LOG_PATH" &
PID=$!
echo "Service starting with PID: $PID"
echo "$PID:1752468175" > "$PID_FILE"
