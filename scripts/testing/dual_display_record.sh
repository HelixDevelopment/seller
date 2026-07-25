#!/bin/bash
# dual_display_record.sh — Screen recording for HelixQA bridge
# Records to MP4 using ffmpeg. Falls back to test pattern when no display.

set -uo pipefail

RECORDING_FILE=""

case "${1:-}" in
    start)
        OUTPUT="${2:-/tmp/__test_recording/hxs_recording_$(date +%s).mp4}"
        mkdir -p "$(dirname "$OUTPUT")"
        echo "Starting recording to $OUTPUT"

        if [ -n "${DISPLAY:-}" ] && command -v ffmpeg >/dev/null 2>&1; then
            # Real screen capture
            ffmpeg -y -f x11grab -r 10 -s 1920x1080 -i :0.0 \
                -c:v libx264 -preset ultrafast -crf 28 \
                -pix_fmt yuv420p "$OUTPUT" &
            RECORDING_PID=$!
            echo "$RECORDING_PID" > /tmp/dual_display_record.pid
            echo "Screen recording started (PID: $RECORDING_PID)"
        elif command -v ffmpeg >/dev/null 2>&1; then
            # Headless: generate a test pattern video
            ffmpeg -y -f lavfi -i testsrc=duration=10:size=1280x720:rate=10 \
                -f lavfi -i anullsrc=r=44100:cl=mono \
                -c:v libx264 -preset ultrafast -crf 28 \
                -c:a aac -shortest "$OUTPUT" &
            RECORDING_PID=$!
            echo "$RECORDING_PID" > /tmp/dual_display_record.pid
            echo "Test pattern recording started (PID: $RECORDING_PID)"
        else
            # No ffmpeg: create placeholder
            echo "NO RECORDING - ffmpeg not installed" > "${OUTPUT}.txt"
            echo "Placeholder recording created (no ffmpeg)"
        fi
        echo "$OUTPUT"
        ;;
    stop)
        if [ -f /tmp/dual_display_record.pid ]; then
            PID=$(cat /tmp/dual_display_record.pid)
            kill "$PID" 2>/dev/null; sleep 1; kill -0 "$PID" 2>/dev/null && kill -9 "$PID" 2>/dev/null || true
            rm -f /tmp/dual_display_record.pid
            echo "Recording stopped"
        else
            echo "No active recording"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop} [output_path]"
        exit 1
        ;;
esac
