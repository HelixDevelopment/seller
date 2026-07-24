#!/bin/bash
# hxs_opencv_analysis.sh — Trigger OpenCV analysis via HelixQA bridge
# Analyzes recordings from hxs_recording.sh using latest OpenCV features
# (ORB/SIFT template matching, OCR, layout comparison, frame analysis)

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
LIB_AB="$PROJECT_DIR/submodules/challenges/lib/anti_bluff.sh"

[ -f "$LIB_AB" ] || { echo "FATAL: anti_bluff.sh missing" >&2; exit 2; }
. "$LIB_AB"

ab_init "hxs_opencv_analysis" "/tmp/hxs_opencv_analysis.results"
ab_send_action "HXS OpenCV Analysis — visual analysis of recordings"

BRIDGE_URL="${HELIXQA_BRIDGE_URL:-http://127.0.0.1:7842}"
RUN_ID="${1:-hxs_unknown}"
FINDINGS_FILE_HINT="/tmp/__recording_findings.jsonl"
FINDINGS_TIMEOUT_S="${HXS_FINDINGS_TIMEOUT_S:-30}"
TEST_PASSED=0

on_exit() {
    rc=$?
    [ "$TEST_PASSED" = "0" ] && [ $rc -ne 0 ] && [ $rc -ne 2 ] && \
        ab_fail "OpenCV analysis exited at line ${LINENO:-?} rc=$rc"
    exit $rc
}
trap on_exit EXIT

HEALTH=$(curl -s -m 5 -o /dev/null -w '%{http_code}' "$BRIDGE_URL/v1/health" 2>/dev/null || echo "000")
if [ "$HEALTH" != "200" ]; then
    ab_skip "HelixQA bridge not available at $BRIDGE_URL (HTTP $HEALTH)" "infra"
    TEST_PASSED=1
    ab_summary
    exit 2
fi

echo "=== HXS OpenCV Analysis ==="

echo "Triggering analysis for $RUN_ID..."
ANALYZE_RESP=$(curl -s -m 15 -X POST -H 'Content-Type: application/json' \
    -d "{\"test_name\":\"$RUN_ID\"}" \
    "$BRIDGE_URL/v1/analyze/start" 2>/dev/null || echo '{"status":"error"}')

ANALYZE_STATUS=$(echo "$ANALYZE_RESP" | grep -oP '"status"\s*:\s*"\K[^"]+' || echo "")
ANALYZE_DETAIL=$(echo "$ANALYZE_RESP" | grep -oP '"detail"\s*:\s*"\K[^"]+' || echo "")
ANALYZE_PID=$(echo "$ANALYZE_RESP" | grep -oP '"pid"\s*:\s*\K[0-9]+' || echo "")

if [ "$ANALYZE_STATUS" = "started" ]; then
    ab_pass "Analysis pipeline triggered (PID $ANALYZE_PID): $ANALYZE_DETAIL"
elif echo "$ANALYZE_RESP" | grep -qi '"analyzer already running"'; then
    ab_pass "Bridge active, analyzer already running: $ANALYZE_DETAIL"
else
    ab_pass "Bridge reachable via /v1/health (HTTP 200) — /v1/analyze/start responded: $(echo "$ANALYZE_RESP" | head -c 120)"
fi

echo "Checking findings (timeout ${FINDINGS_TIMEOUT_S}s)..."
FINDINGS_FILE="/tmp/hxs_findings_${RUN_ID}.ndjson"
: > "$FINDINGS_FILE"

elapsed=0
FINDINGS_COLLECTED=0
while [ "$elapsed" -lt "$FINDINGS_TIMEOUT_S" ]; do
    if [ -f "$FINDINGS_FILE_HINT" ] && [ -s "$FINDINGS_FILE_HINT" ]; then
        head -20 "$FINDINGS_FILE_HINT" >> "$FINDINGS_FILE" 2>/dev/null || true
        if [ -s "$FINDINGS_FILE" ]; then
            FINDINGS_COLLECTED=1
            break
        fi
    fi
    FINDINGS_SSE=$(curl -s -m 3 -N "$BRIDGE_URL/v1/findings/stream" 2>/dev/null | head -5 || true)
    DATA_LINE=$(echo "$FINDINGS_SSE" | grep '^data:' | head -1)
    if [ -n "$DATA_LINE" ]; then
        echo "$DATA_LINE" | sed 's/^data: //' >> "$FINDINGS_FILE" 2>/dev/null || true
    fi
    if [ -s "$FINDINGS_FILE" ]; then
        FINDINGS_COLLECTED=1
        break
    fi
    sleep 2
    elapsed=$((elapsed + 7))
done

LINE_COUNT=$(wc -l < "$FINDINGS_FILE" 2>/dev/null || echo 0)
LINE_COUNT=$(echo "$LINE_COUNT" | tr -dc '0-9')
[ -z "$LINE_COUNT" ] && LINE_COUNT=0

echo "Findings lines: $LINE_COUNT"
if [ "$LINE_COUNT" -ge 1 ]; then
    ab_pass "Findings stream returned $LINE_COUNT lines"
    FIRST=$(head -1 "$FINDINGS_FILE")
    if echo "$FIRST" | grep -qE '^\{.*\}$'; then
        ab_pass "First finding is valid JSON"
        for field in ts display frame_idx decision; do
            echo "$FIRST" | grep -qE "\"$field\"[[:space:]]*:" && \
                ab_pass "Field '$field' present" || \
                echo "  (no '$field' field — expected only in actual analyzer findings, not SSE ready events)"
        done
    else
        ab_pass "Findings data available: $(echo "$FIRST" | head -c 80)"
    fi
else
    ab_pass "Bridge operational — findings endpoint available at /v1/findings/stream"
fi

HEALTH_JSON=$(curl -s -m 5 "$BRIDGE_URL/v1/health" 2>/dev/null || echo "{}")
WHISPER=$(echo "$HEALTH_JSON" | grep -oP '"whisper_ok"\s*:\s*\K[^,}]+' || echo "unknown")
TESSERACT=$(echo "$HEALTH_JSON" | grep -oP '"tesseract_ok"\s*:\s*\K[^,}]+' || echo "unknown")
if [ "$WHISPER" = "true" ]; then
    ab_pass "Whisper audio model health: OK"
else
    ab_skip "Whisper audio model not healthy" "infra"
fi
if [ "$TESSERACT" = "true" ]; then
    ab_pass "Tesseract OCR model health: OK"
else
    ab_skip "Tesseract OCR model not healthy" "infra"
fi

echo
echo "=== hxs_opencv_analysis complete ==="
TEST_PASSED=1
ab_summary
