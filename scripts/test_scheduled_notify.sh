#!/usr/bin/env bash
# 定时每小时系统通知联调（手动触发 broadcast + sync 验证）。
# 前置：my_go_study/ 目录下 make run + make run-worker，且用户已登录（有 auth:session）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
EMAIL="${TEST_EMAIL:-demo@example.com}"
PASSWORD="${TEST_PASSWORD:-123456}"
DEVICE_ID="${TEST_DEVICE_ID:-sched-test-device-$(date +%s)}"
TOKEN="${SUPABASE_ACCESS_TOKEN:-}"
SESSION_ID=""

check_server() {
  if ! curl -sf --connect-timeout 3 "$BASE_URL/health" >/dev/null; then
    echo "错误: $BASE_URL/health 不可达"
    echo "  终端 1: cd my_go_study && make run"
    echo "  终端 2: cd my_go_study && make run-worker"
    exit 1
  fi
}

login() {
  echo ">>> 1. login via BFF ($EMAIL) — 创建 auth:session"
  LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"device_id\":\"$DEVICE_ID\",\"platform\":\"ios\"}")
  TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN_RESP")
  SESSION_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$LOGIN_RESP")
  echo "    session_id=$SESSION_ID"
}

trigger_broadcast() {
  echo ">>> 2. make trigger-hourly-notify（入队 scheduled:broadcast_notify）"
  ./scripts/load-env.sh go run ./cmd/scheduler-trigger
  echo ">>> 3. wait worker broadcast + per-user push"
  sleep 3
}

verify_sync() {
  echo ">>> 4. POST /api/v1/realtime/sync"
  SYNC_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/realtime/sync" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Session-ID: $SESSION_ID" \
    -H "X-Device-ID: $DEVICE_ID" \
    -H "Content-Type: application/json" \
    -d '{"sinceSeq":0,"topics":["sys.notify"]}')
  python3 -c "
import json,sys
data=json.load(sys.stdin)['data']
events=data.get('events') or []
if not events:
    raise SystemExit('sync 未拉到 scheduled 通知，请确认 Worker 已运行且 auth:session 存在')
last=events[-1]['payload']
assert last.get('category')=='scheduled', last
assert last.get('campaignId'), last
print('    events=', len(events), 'last campaignId=', last.get('campaignId'))
" <<<"$SYNC_RESP"
}

check_server
if [[ -z "$TOKEN" ]]; then
  login
fi
trigger_broadcast
verify_sync
echo ">>> 定时广播联调通过"
