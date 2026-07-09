#!/usr/bin/env bash
# Realtime WebSocket MVP 联调脚本（需 Go 后端 + Redis + Supabase JWT）。
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
TOKEN="${SUPABASE_ACCESS_TOKEN:-}"
SERVICE_ROLE="${SUPABASE_SERVICE_ROLE_KEY:-}"

ensure_token() {
  if [[ -n "$TOKEN" ]]; then
    return 0
  fi

  echo ">>> 2. login via BFF ($EMAIL)"
  if LOGIN_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" 2>/dev/null); then
    TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN_RESP")
    USER_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['user']['id'])" <<<"$LOGIN_RESP")
    return 0
  fi

  if [[ -z "$SERVICE_ROLE" ]]; then
    echo "错误: 登录失败，且未配置 SUPABASE_ACCESS_TOKEN / SUPABASE_SERVICE_ROLE_KEY"
    echo "  export SUPABASE_ACCESS_TOKEN='eyJ...'"
    echo "  或在 .env.local 配置 SUPABASE_SERVICE_ROLE_KEY 后重试"
    exit 1
  fi

  EMAIL="rt_ws_$(date +%s)@gmail.com"
  PASSWORD="TestPass123!"
  echo ">>> 2. 使用 service_role 创建已确认测试用户: $EMAIL"
  curl -sf --connect-timeout 15 --max-time 30 \
    -X POST "${SUPABASE_URL}/auth/v1/admin/users" \
    -H "apikey: ${SERVICE_ROLE}" \
    -H "Authorization: Bearer ${SERVICE_ROLE}" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"email_confirm\":true}" >/dev/null

  local login_resp
  login_resp=$(curl -sf --connect-timeout 15 --max-time 30 \
    -X POST "${SUPABASE_URL}/auth/v1/token?grant_type=password" \
    -H "apikey: ${ANON_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")
  TOKEN=$(echo "$login_resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))")
  USER_ID=$(echo "$login_resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('user',{}).get('id',''))")
  if [[ -z "$TOKEN" ]]; then
    echo "登录失败: $login_resp"
    exit 1
  fi
}

echo ">>> 1. health"
curl -sf "$BASE_URL/health" | grep -q ok && echo "OK"

if [[ -n "$TOKEN" ]]; then
  echo ">>> 2. 使用 SUPABASE_ACCESS_TOKEN"
  USER_ID="${TEST_USER_ID:-}"
else
  ensure_token
fi
echo "userId=${USER_ID:-<from token>} token_len=${#TOKEN}"

echo ">>> 3. ws-ticket"
TICKET_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/realtime/ws-ticket" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"platform":"mobile"}')
echo "$TICKET_RESP"
TICKET=$(python3 -c "import json,sys; print(json.load(sys.stdin)['ticket'])" <<<"$TICKET_RESP")
WS_URL=$(python3 -c "import json,sys; print(json.load(sys.stdin)['wsUrl'])" <<<"$TICKET_RESP")
CONN_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['connId'])" <<<"$TICKET_RESP")

echo ">>> 4. push notify"
PUSH_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/realtime/push" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"联调通知","body":"来自 scripts/test_realtime_ws.sh"}')
echo "$PUSH_RESP"

echo ">>> 5. sync"
SYNC_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/realtime/sync" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sinceSeq":0,"topics":["sys.notify"]}')
echo "$SYNC_RESP"

echo ">>> 6. websocket auth (go test)"
export TEST_WS_URL="$WS_URL" TEST_WS_TICKET="$TICKET" TEST_WS_CONN_ID="$CONN_ID"
go test ./internal/delivery/ws/ -run TestWSAuthFlow -count=1 -v

echo ""
echo "✅ Realtime MVP 联调通过"
