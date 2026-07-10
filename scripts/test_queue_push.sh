#!/usr/bin/env bash
# 异步 Push（Asynq + Pub/Sub）联调脚本。
# 前置：make run（API）+ make run-worker（Worker），queue.enabled=true（config.dev.yaml 默认开启）。
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
DEVICE_ID="${TEST_DEVICE_ID:-queue-test-device-$(date +%s)}"
TOKEN="${SUPABASE_ACCESS_TOKEN:-}"
SERVICE_ROLE="${SUPABASE_SERVICE_ROLE_KEY:-}"
SESSION_ID=""

check_server() {
  if ! curl -sf --connect-timeout 3 "$BASE_URL/health" >/dev/null; then
    echo "错误: $BASE_URL/health 不可达"
    echo "  终端 1: make run"
    echo "  终端 2: make run-worker"
    exit 1
  fi
}

ensure_token() {
  echo ">>> 1. login via BFF ($EMAIL)"
  if LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"device_id\":\"$DEVICE_ID\",\"platform\":\"ios\"}" 2>/dev/null); then
    TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN_RESP")
    SESSION_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$LOGIN_RESP")
    echo "    token ok, session_id=$SESSION_ID"
    return 0
  fi

  if [[ -z "$SERVICE_ROLE" ]]; then
    echo "错误: 登录失败（demo@example.com 可能未注册），且未配置 SUPABASE_SERVICE_ROLE_KEY"
    echo "  在 .env.local 配置 SUPABASE_SERVICE_ROLE_KEY 后重试，或："
    echo "  export TEST_EMAIL='你的邮箱' TEST_PASSWORD='你的密码'"
    exit 1
  fi

  EMAIL="queue_push_$(date +%s)@gmail.com"
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
  echo "    token ok, session_id=$SESSION_ID"
}

enqueue_push() {
  echo ">>> 2. POST /api/v1/realtime/push (async enqueue)"
  PUSH_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/realtime/push" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Session-ID: $SESSION_ID" \
    -H "X-Device-ID: $DEVICE_ID" \
    -H "Content-Type: application/json" \
    -d '{"title":"Queue Test","body":"async push via Asynq"}')
  echo "$PUSH_RESP" | python3 -c "
import json,sys
d=json.load(sys.stdin)['data']
assert d.get('queued') is True, d
assert d.get('taskId'), d
print('    queued=true taskId=', d['taskId'])
"

  echo ">>> 3. wait worker delivery + sync"
  sleep 2
  SYNC_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/realtime/sync" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Session-ID: $SESSION_ID" \
    -H "X-Device-ID: $DEVICE_ID" \
    -H "Content-Type: application/json" \
    -d '{"sinceSeq":0,"topics":["sys.notify"]}')
  COUNT=$(python3 -c "import json,sys; print(len(json.load(sys.stdin).get('events') or []))" <<<"$SYNC_RESP")
  if [[ "$COUNT" -lt 1 ]]; then
    echo "错误: sync 未拉到事件，请确认 Worker 已启动且 queue.enabled=true"
    echo "$SYNC_RESP"
    exit 1
  fi
  echo "    sync events=$COUNT OK"
}

check_server
if [[ -z "$TOKEN" || -z "$SESSION_ID" ]]; then
  ensure_token
fi
enqueue_push
echo ">>> 异步 Push 联调通过"
