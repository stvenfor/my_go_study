#!/usr/bin/env bash
# transactions CRUD 联调脚本
# 用法:
#   export SUPABASE_ACCESS_TOKEN="<Flutter 登录后的 access_token>"
#   ./scripts/test_transactions_crud.sh
#
# 或配置 service_role 自动创建测试账号:
#   export SUPABASE_SERVICE_ROLE_KEY="<Dashboard → API → service_role>"
#   ./scripts/test_transactions_crud.sh

set -euo pipefail

# 自动加载 .env + .env.local
# shellcheck disable=SC1091
source "$(dirname "$0")/source-env.sh"

BASE_URL="${BASE_URL:-http://localhost:8080}"
SUPABASE_URL="${SUPABASE_URL:?请在 configs/supabase.env 配置 SUPABASE_URL}"
ANON_KEY="${SUPABASE_ANON_KEY:?请在 configs/supabase.env 配置 SUPABASE_ANON_KEY}"
TOKEN="${SUPABASE_ACCESS_TOKEN:-}"
SERVICE_ROLE="${SUPABASE_SERVICE_ROLE_KEY:-}"
DEVICE_ID="${TEST_DEVICE_ID:-tx-crud-device-$(date +%s)}"
SESSION_ID=""
EMAIL="${TEST_EMAIL:-}"
PASSWORD="${TEST_PASSWORD:-123456}"

auth_headers=(
  -H "Authorization: Bearer ${TOKEN}"
  -H "X-Session-ID: ${SESSION_ID}"
  -H "X-Device-ID: ${DEVICE_ID}"
)

refresh_auth_headers() {
  auth_headers=(
    -H "Authorization: Bearer ${TOKEN}"
    -H "X-Session-ID: ${SESSION_ID}"
    -H "X-Device-ID: ${DEVICE_ID}"
  )
}

bff_login() {
  local email="$1"
  local password="$2"
  local resp
  resp=$(curl -sf -X POST "${BASE_URL}/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${email}\",\"password\":\"${password}\",\"device_id\":\"${DEVICE_ID}\",\"platform\":\"ios\"}")
  TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$resp")
  SESSION_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$resp")
  refresh_auth_headers
}

ensure_token() {
  if [[ -n "$TOKEN" && -n "$SESSION_ID" ]]; then
    return 0
  fi
  if [[ -n "$EMAIL" ]]; then
    echo ">>> 通过 BFF 登录: $EMAIL"
    bff_login "$EMAIL" "$PASSWORD"
    echo ">>> token 已获取 (长度 ${#TOKEN}) session_id=${SESSION_ID}"
    return 0
  fi
  if [[ -z "$SERVICE_ROLE" ]]; then
    echo "错误: 需要 TEST_EMAIL+密码、SUPABASE_ACCESS_TOKEN+SESSION，或 SUPABASE_SERVICE_ROLE_KEY"
    echo ""
    echo "方式 1 — 指定测试账号:"
    echo "  export TEST_EMAIL='demo@example.com' TEST_PASSWORD='123456'"
    echo ""
    echo "方式 2 — 配置 service_role 自动创建测试用户:"
    echo "  export SUPABASE_SERVICE_ROLE_KEY='eyJ...'"
    exit 1
  fi

  EMAIL="go_tx_crud_$(date +%s)@gmail.com"
  PASSWORD="TestPass123!"

  echo ">>> 使用 service_role 创建已确认测试用户: $EMAIL"
  curl -sf --connect-timeout 15 --max-time 30 \
    -X POST "${SUPABASE_URL}/auth/v1/admin/users" \
    -H "apikey: ${SERVICE_ROLE}" \
    -H "Authorization: Bearer ${SERVICE_ROLE}" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"email_confirm\":true}" >/dev/null

  echo ">>> 通过 BFF 登录获取 token + session"
  bff_login "$EMAIL" "$PASSWORD"
  if [[ -z "$TOKEN" || -z "$SESSION_ID" ]]; then
    echo "BFF 登录失败"
    exit 1
  fi
  echo ">>> token 已获取 (长度 ${#TOKEN}) session_id=${SESSION_ID}"
}

check_server() {
  if ! curl -sf --connect-timeout 3 "${BASE_URL}/health" >/dev/null; then
    echo "错误: ${BASE_URL}/health 不可达，请先 make run"
    exit 1
  fi
}

pretty() {
  python3 -m json.tool 2>/dev/null || cat
}

echo "========================================"
echo " transactions CRUD 联调"
echo " BASE_URL=${BASE_URL}"
echo "========================================"

check_server
ensure_token

echo ""
echo ">>> [0] 无 token 应返回 401"
code=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/api/v1/transactions")
echo "GET /api/v1/transactions (无 Auth) => HTTP ${code}"

echo ""
echo ">>> [1] GET /api/v1/transactions?limit=3"
GET_LIST=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
  "${auth_headers[@]}" \
  "${BASE_URL}/api/v1/transactions?limit=3")
HTTP_CODE=$(echo "$GET_LIST" | grep '__HTTP_CODE__' | cut -d: -f2)
BODY=$(echo "$GET_LIST" | sed '/__HTTP_CODE__/d')
echo "HTTP ${HTTP_CODE}"
echo "$BODY" | pretty

echo ""
echo ">>> [2] POST /api/v1/transactions"
CREATE_BODY='{"type":"expense","category":"餐饮","amount":88.5,"date":"2026-07-08","note":"CRUD测试创建"}'
CREATE_RESP=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
  -X POST "${BASE_URL}/api/v1/transactions" \
  "${auth_headers[@]}" \
  -H "Content-Type: application/json" \
  -d "${CREATE_BODY}")
HTTP_CODE=$(echo "$CREATE_RESP" | grep '__HTTP_CODE__' | cut -d: -f2)
BODY=$(echo "$CREATE_RESP" | sed '/__HTTP_CODE__/d')
echo "HTTP ${HTTP_CODE}"
echo "$BODY" | pretty

TX_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || true)
if [[ -z "$TX_ID" ]]; then
  echo "创建失败，跳过后续 PUT/DELETE"
  exit 1
fi
echo ">>> 新建记录 id=${TX_ID}"

echo ""
echo ">>> [3] GET /api/v1/transactions/${TX_ID}"
GET_ONE=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
  "${auth_headers[@]}" \
  "${BASE_URL}/api/v1/transactions/${TX_ID}")
HTTP_CODE=$(echo "$GET_ONE" | grep '__HTTP_CODE__' | cut -d: -f2)
BODY=$(echo "$GET_ONE" | sed '/__HTTP_CODE__/d')
echo "HTTP ${HTTP_CODE}"
echo "$BODY" | pretty

echo ""
echo ">>> [4] PUT /api/v1/transactions/${TX_ID}"
UPDATE_BODY='{"category":"交通","amount":99.9,"note":"CRUD测试更新"}'
UPDATE_RESP=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
  -X PUT "${BASE_URL}/api/v1/transactions/${TX_ID}" \
  "${auth_headers[@]}" \
  -H "Content-Type: application/json" \
  -d "${UPDATE_BODY}")
HTTP_CODE=$(echo "$UPDATE_RESP" | grep '__HTTP_CODE__' | cut -d: -f2)
BODY=$(echo "$UPDATE_RESP" | sed '/__HTTP_CODE__/d')
echo "HTTP ${HTTP_CODE}"
echo "$BODY" | pretty

echo ""
echo ">>> [5] DELETE /api/v1/transactions/${TX_ID}"
DELETE_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -X DELETE "${BASE_URL}/api/v1/transactions/${TX_ID}" \
  $(auth_headers))
echo "HTTP ${DELETE_CODE}"

echo ""
echo ">>> [6] 再次 GET 应 404"
GET_GONE=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
  "${auth_headers[@]}" \
  "${BASE_URL}/api/v1/transactions/${TX_ID}")
HTTP_CODE=$(echo "$GET_GONE" | grep '__HTTP_CODE__' | cut -d: -f2)
BODY=$(echo "$GET_GONE" | sed '/__HTTP_CODE__/d')
echo "HTTP ${HTTP_CODE}"
echo "$BODY" | pretty

echo ""
echo "========================================"
echo " 联调完成"
echo "========================================"
