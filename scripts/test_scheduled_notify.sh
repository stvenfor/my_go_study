#!/usr/bin/env bash
# 定时每小时系统通知联调（手动触发 broadcast + sync 验证）。
# 前置：my_go_study/ 目录下 make run + make run-worker，且用户已登录（有 auth:session）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SUPABASE_URL="${SUPABASE_URL:?请在 configs/supabase.env 配置 SUPABASE_URL}"
ANON_KEY="${SUPABASE_ANON_KEY:?请在 configs/supabase.env 配置 SUPABASE_ANON_KEY}"
EMAIL="${TEST_EMAIL:-demo@example.com}"
PASSWORD="${TEST_PASSWORD:-123456}"
DEVICE_ID="${TEST_DEVICE_ID:-sched-test-device-$(date +%s)}"
TOKEN="${SUPABASE_ACCESS_TOKEN:-}"
SERVICE_ROLE="${SUPABASE_SERVICE_ROLE_KEY:-}"
SESSION_ID=""

check_server() {
  if ! curl -sf --connect-timeout 3 "$BASE_URL/health" >/dev/null; then
    echo "错误: $BASE_URL/health 不可达"
    echo "  终端 1: cd my_go_study && make run"
    echo "  终端 2: cd my_go_study && make run-worker"
    exit 1
  fi
}

ensure_token() {
  echo ">>> 1. login via BFF ($EMAIL) — 创建 auth:session"
  if LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"device_id\":\"$DEVICE_ID\",\"platform\":\"ios\"}" 2>/dev/null); then
    TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN_RESP")
    SESSION_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$LOGIN_RESP")
    echo "    session_id=$SESSION_ID"
    return 0
  fi

  if [[ -z "$SERVICE_ROLE" ]]; then
    echo "错误: 登录失败（demo@example.com 可能未注册），且未配置 SUPABASE_SERVICE_ROLE_KEY"
    echo "  在 .env.local 配置 SUPABASE_SERVICE_ROLE_KEY 后重试，或："
    echo "  export TEST_EMAIL='你的邮箱' TEST_PASSWORD='你的密码'"
    exit 1
  fi

  EMAIL="sched_notify_$(date +%s)@gmail.com"
  PASSWORD="TestPass123!"
  echo ">>> 1. 使用 service_role 创建已确认测试用户: $EMAIL"
  curl -sf --connect-timeout 15 --max-time 30 \
    -X POST "${SUPABASE_URL}/auth/v1/admin/users" \
    -H "apikey: ${SERVICE_ROLE}" \
    -H "Authorization: Bearer ${SERVICE_ROLE}" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"email_confirm\":true}" >/dev/null

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
  sleep 4
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
body=json.load(sys.stdin)
events=body.get('events') or []
if not events:
    raise SystemExit('sync 未拉到 scheduled 通知，请确认 Worker 已运行且 auth:session 存在')
last=events[-1]['payload']
assert last.get('category')=='scheduled', last
assert last.get('campaignId'), last
print('    events=', len(events), 'last campaignId=', last.get('campaignId'))
" <<<"$SYNC_RESP"
}

check_server
if [[ -z "$TOKEN" || -z "$SESSION_ID" ]]; then
  ensure_token
fi
trigger_broadcast
verify_sync
echo ">>> 定时广播联调通过"
