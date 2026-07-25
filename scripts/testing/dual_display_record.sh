#!/bin/bash
# dual_display_record.sh — Screen recording via Playwright + ffmpeg
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REC_DIR="${REC_DIR:-/tmp/__test_recording}"
PID_FILE="/tmp/dual_display_record.pid"

case "${1:-}" in
    start)
        TEST_NAME="${2:-unknown}"
        mkdir -p "$REC_DIR"
        SHOT_DIR="$REC_DIR/${TEST_NAME}_shots"
        rm -rf "$SHOT_DIR"
        mkdir -p "$SHOT_DIR"

        OUTPUT="$REC_DIR/${TEST_NAME}_primary.mp4"
        echo "$OUTPUT"

        WEB_DIR="$(cd "$SCRIPT_DIR/../../web" && pwd)"
        NODE_PATH="$WEB_DIR/node_modules" node "$SCRIPT_DIR/dual_display_record.js" start "$TEST_NAME" "$SHOT_DIR" &
        PID=$!
        echo "$PID" > "$PID_FILE"
        ;;
    stop)
        TEST_NAME="${2:-unknown}"
        if [ -f "$PID_FILE" ]; then
            CPID=$(cat "$PID_FILE")
            SHOT_DIR="$REC_DIR/${TEST_NAME}_shots"
            VIDEO="$REC_DIR/${TEST_NAME}_primary.mp4"
            WEB_DIR="$(cd "$SCRIPT_DIR/../../web" && pwd)"

            NODE_PATH="$WEB_DIR/node_modules" timeout 90 node "$SCRIPT_DIR/dual_display_record.js" stop "$TEST_NAME" 2>/dev/null
            rm -f "$PID_FILE"

            [ -f "$VIDEO" ] && echo "$(stat -c%s "$VIDEO")"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop} [test_name]"
        exit 1
        ;;
esac
