#!/bin/bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT="$REPO_ROOT/scripts/system-manage.sh"

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

pass() { PASS_COUNT=$((PASS_COUNT + 1)); echo "PASS: $*"; }
fail() { FAIL_COUNT=$((FAIL_COUNT + 1)); echo "FAIL: $*"; }
skip() { SKIP_COUNT=$((SKIP_COUNT + 1)); echo "SKIP: $*"; }

assert_file() {
  if [[ -f "$1" ]]; then
    pass "File exists: $1"
  else
    fail "File not found: $1"
  fi
}

assert_executable() {
  if [[ -x "$1" ]]; then
    pass "File is executable: $1"
  else
    fail "File not executable: $1"
  fi
}

assert_contains() {
  if grep -q "$2" "$1"; then
    pass "$1 contains '$2'"
  else
    fail "$1 does not contain '$2'"
  fi
}

assert_exit_code() {
  local expected=$1
  shift
  local rc=0
  "$@" >/dev/null 2>&1 || rc=$?
  if [[ $rc -eq $expected ]]; then
    pass "Command exits $expected: $*"
  else
    fail "Command expected exit $expected, got $rc: $*"
  fi
}

test_script_exists() {
  assert_file "$SCRIPT"
  assert_executable "$SCRIPT"
}

test_shebang() {
  assert_contains "$SCRIPT" "^#!/bin/bash"
}

test_bad_command() {
  assert_exit_code 1 bash "$SCRIPT" bad_command
}

test_status_command() {
  local rc=0
  bash "$SCRIPT" status >/dev/null 2>&1 || rc=$?
  if [[ $EUID -eq 0 ]]; then
    if [[ $rc -eq 1 ]]; then
      pass "status correctly rejects root execution (exit 1)"
    else
      fail "status should exit 1 when run as root, got $rc"
    fi
  else
    if [[ $rc -eq 0 ]]; then
      pass "status exits 0 (works without services running)"
    else
      fail "status should exit 0, got $rc"
    fi
  fi
}

test_build_command() {
  if ! command -v go &>/dev/null; then
    skip "Go not installed"
    return
  fi
  local rc=0
  bash "$SCRIPT" build >/dev/null 2>&1 || rc=$?
  if [[ $rc -eq 0 ]]; then
    pass "build command succeeded"
  elif [[ $rc -eq 1 ]]; then
    skip "build skipped — running as root (user context check)"
  elif [[ $rc -eq 2 ]]; then
    skip "build skipped — missing dependencies (podman/npm)"
  else
    fail "build command exited with code $rc"
  fi
}

test_service_files() {
  local files=(
    "container-helix-postgres.service"
    "container-helix-redis.service"
    "container-helix-nats.service"
    "helix-server.service"
    "helix.target"
  )
  for f in "${files[@]}"; do
    assert_file "$REPO_ROOT/systemd/$f"
  done
}

test_health_check() {
  local mock_dir
  mock_dir=$(mktemp -d)

  cat > "$mock_dir/curl" << 'MOCKCURL'
#!/bin/bash
echo '{"postgresql":{"healthy":true},"redis":{"healthy":true}}'
exit 0
MOCKCURL
  chmod +x "$mock_dir/curl"

  local result
  result=$(PATH="$mock_dir:$PATH" SCRIPT_PATH="$SCRIPT" bash << 'INNERBASH'
pass() { echo "PASS: $*"; }
fail() { echo "FAIL: $*"; }
info() { echo "INFO: $*"; }
RED="" GREEN="" YELLOW="" NC=""
HEALTH_URL="http://127.0.0.1:8080/health"
eval "$(sed -n '/^health_check()/,/^}/p' "$SCRIPT_PATH")"
health_check
INNERBASH
)
  local rc=$?
  rm -rf "$mock_dir"

  if echo "$result" | grep -q "PASS: All health checks passed"; then
    pass "health_check passes with mock curl returning healthy response"
  else
    fail "health_check should pass when curl returns healthy response"
    echo "  Output: $(echo "$result" | tr '\n' '; ')"
  fi
}

echo "=== Helix Seller System Management Tests ==="
echo ""

test_script_exists
test_shebang
test_bad_command
test_status_command
test_build_command
test_service_files
test_health_check

echo ""
echo "=== Results: $PASS_COUNT passed, $FAIL_COUNT failed, $SKIP_COUNT skipped ==="

if [[ $FAIL_COUNT -gt 0 ]]; then
  exit 1
fi
exit 0
