#!/bin/bash
# Helix Seller — Master System Management Script
set -euo pipefail

PROJECT_DIR="$(git rev-parse --show-toplevel 2>/dev/null)"
SYSTEMD_DIR="${PROJECT_DIR}/systemd"
USER_SYSTEMD_DIR="${HOME}/.config/systemd/user"
TARGET="helix.target"
BUILD_DIR="${PROJECT_DIR}/build"
BINARY="${BUILD_DIR}/helix-server"
HEALTH_URL="http://127.0.0.1:8080/health"

SERVICES=(
  "container-helix-postgres.service"
  "container-helix-redis.service"
  "container-helix-nats.service"
  "helix-server.service"
  "helix-angular.service"
)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { printf "${GREEN}PASS${NC} %s\n" "$*"; }
fail() { printf "${RED}FAIL${NC} %s\n" "$*"; }
skip() { printf "${YELLOW}SKIP${NC} %s\n" "$*"; }
info() { printf "INFO %s\n" "$*"; }

cleanup() {
  :
}
trap cleanup EXIT

ensure_user_context() {
  if [[ $EUID -eq 0 ]]; then
    fail "Do not run this script as root. Use a regular user with --user systemd."
    exit 1
  fi
}

ensure_user_systemd_dir() {
  mkdir -p "$USER_SYSTEMD_DIR"
}

check_prereqs() {
  local missing=0
  for cmd in podman go npm; do
    if ! command -v "$cmd" &>/dev/null; then
      fail "Prerequisite not found: $cmd"
      missing=1
    fi
  done
  if [[ $missing -eq 1 ]]; then
    exit 2
  fi
  pass "All prerequisites found"
}

install_services() {
  ensure_user_systemd_dir
  info "Copying service files to ${USER_SYSTEMD_DIR}..."
  cp "$SYSTEMD_DIR"/*.service "$USER_SYSTEMD_DIR"/ 2>/dev/null || true
  cp "$SYSTEMD_DIR"/*.target "$USER_SYSTEMD_DIR"/ 2>/dev/null || true
  systemctl --user daemon-reload
  systemctl --user enable "$TARGET"
  pass "Services installed and enabled"
}

cmd_install() {
  ensure_user_context
  check_prereqs
  install_services
}

cmd_start() {
  ensure_user_context
  check_prereqs
  info "Starting ${TARGET}..."
  systemctl --user start "$TARGET" 2>/dev/null || {
    fail "Failed to start ${TARGET}. Starting services individually..."
    for svc in "${SERVICES[@]}"; do
      systemctl --user start "$svc" 2>/dev/null || fail "Could not start $svc"
    done
  }
  sleep 3
  health_check
}

cmd_stop() {
  ensure_user_context
  info "Stopping ${TARGET}..."
  systemctl --user stop "$TARGET" 2>/dev/null || true
  for svc in "${SERVICES[@]}"; do
    systemctl --user stop "$svc" 2>/dev/null || true
  done
  pass "All services stopped"
}

cmd_restart() {
  cmd_stop
  sleep 2
  cmd_start
}

cmd_status() {
  ensure_user_context
  local all_active=true
  for svc in "${SERVICES[@]}"; do
    if systemctl --user is-active --quiet "$svc" 2>/dev/null; then
      pass "$svc"
    else
      fail "$svc"
      all_active=false
    fi
  done
  if systemctl --user is-active --quiet "$TARGET" 2>/dev/null; then
    pass "${TARGET}"
  else
    fail "${TARGET}"
  fi
  echo ""
  if $all_active; then
    pass "All services running"
  else
    fail "Some services not running"
  fi
}

cmd_logs() {
  ensure_user_context
  if [[ $# -eq 0 ]]; then
    info "Showing logs for all services (last 50 lines each):"
    echo ""
    for svc in "${SERVICES[@]}"; do
      echo "=== $svc ==="
      journalctl --user -u "$svc" --no-pager -n 50 2>/dev/null || skip "No logs for $svc"
      echo ""
    done
  else
    local service="$1"
    if [[ "$service" != *.* ]]; then
      service="${service}.service"
    fi
    journalctl --user -u "$service" --no-pager -n 50 2>/dev/null || fail "No logs for $service"
  fi
}

cmd_build() {
  ensure_user_context
  check_prereqs
  info "Building Go binary to ${BINARY}..."
  mkdir -p "$BUILD_DIR"
  (cd "$PROJECT_DIR" && go build -o "$BINARY" ./cmd/server/)
  pass "Binary built at ${BINARY}"
}

cmd_clean() {
  ensure_user_context
  info "Stopping all services..."
  cmd_stop
  info "Removing Podman containers..."
  for container in helix-postgres helix-redis helix-nats; do
    podman rm -f "$container" 2>/dev/null || true
  done
  info "Cleaning Podman volumes..."
  for vol in helix-postgres-data helix-nats-data; do
    podman volume rm "$vol" 2>/dev/null || true
  done
  info "Cleaning build artifacts..."
  rm -rf "$BUILD_DIR"
  pass "Clean complete"
}

health_check() {
  local healthy=true
  info "Running health checks..."

  if curl -sf "$HEALTH_URL" >/dev/null 2>&1; then
    pass "Server health endpoint reachable"
    local health_json
    health_json=$(curl -sf "$HEALTH_URL" 2>/dev/null || echo "")
    if echo "$health_json" | grep -q '"postgresql"\s*:\s*{[^}]*"healthy"\s*:\s*true'; then
      pass "PostgreSQL health check passed"
    else
      if pg_isready -h 127.0.0.1 -p 5432 -U helix &>/dev/null; then
        pass "PostgreSQL direct check passed"
      else
        fail "PostgreSQL health check failed"
        healthy=false
      fi
    fi
    if echo "$health_json" | grep -q '"redis"\s*:\s*{[^}]*"healthy"\s*:\s*true'; then
      pass "Redis health check passed"
    else
      fail "Redis health check failed (no health endpoint confirmation)"
      healthy=false
    fi
  else
    fail "Server health endpoint not reachable"
    info "Checking services directly..."
    if pg_isready -h 127.0.0.1 -p 5432 -U helix &>/dev/null; then
      pass "PostgreSQL is accepting connections"
    else
      fail "PostgreSQL is not accepting connections"
      healthy=false
    fi
    if echo "PING" | redis-cli -h 127.0.0.1 2>/dev/null | grep -q "PONG"; then
      pass "Redis is responding"
    else
      fail "Redis is not responding"
      healthy=false
    fi
  fi

  if $healthy; then
    pass "All health checks passed"
  else
    fail "Some health checks failed"
  fi
}

usage() {
  echo "Usage: $0 <command> [args]"
  echo ""
  echo "Commands:"
  echo "  install           Install systemd user services, enable helix.target"
  echo "  start             Start all services via helix.target"
  echo "  stop              Stop all services"
  echo "  restart           Restart all services"
  echo "  status            Show status of all services"
  echo "  logs [service]    Show logs for a specific service (or all)"
  echo "  build             Build the Go binary to build/helix-server"
  echo "  clean             Stop all services, remove containers, clean data"
  echo "  help              Show this help message"
}

case "${1:-help}" in
  install)
    cmd_install
    ;;
  start)
    cmd_start
    ;;
  stop)
    cmd_stop
    ;;
  restart)
    cmd_restart
    ;;
  status)
    cmd_status
    ;;
  logs)
    shift
    cmd_logs "$@"
    ;;
  build)
    cmd_build
    ;;
  clean)
    cmd_clean
    ;;
  help|--help|-h)
    usage
    ;;
  *)
    fail "Unknown command: $1"
    usage
    exit 1
    ;;
esac
