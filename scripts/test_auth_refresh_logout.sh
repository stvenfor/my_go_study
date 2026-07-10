#!/usr/bin/env bash
# refresh / logout 联调：登录 → refresh → 业务 API → logout → 401
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SUPABASE_URL="${SUPABASE_URL:-}"
EMAIL="${TEST_EMAIL:-demo@example.com}"
PASSWORD="${TEST_PASSWORD:-123456}"
DEVICE_ID="${TEST_DEVICE_ID:-refresh-test-$(date +%s)}"
SERVICE_ROLE="${SUPABASE_SERVICE_ROLE_KEY:-}"

check_server() {
  if ! curl -sf --connect-timeout 3 "$BASE_URL/health" >/dev/null; then
    echo "错误: $BASE_URL/health 不可达，请先 make run" >&2
    exit 1
  fi
}

ensure_login() {
  if LOGIN=$(curl -sf -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"device_id\":\"$DEVICE_ID\",\"platform\":\"ios\"}" 2>/dev/null); then
    printf '%s' "$LOGIN"
    return 0
  fi

  if [[ -z "$SERVICE_ROLE" || -z "$SUPABASE_URL" ]]; then
    echo "错误: 登录失败，请配置 SUPABASE_SERVICE_ROLE_KEY 或 export TEST_EMAIL/TEST_PASSWORD" >&2
    exit 1
  fi

  EMAIL="auth_refresh_$(date +%s)@gmail.com"
  PASSWORD="TestPass123!"
  echo ">>> 创建测试用户 $EMAIL" >&2
  curl -sf --connect-timeout 15 --max-time 30 \
    -X POST "${SUPABASE_URL}/auth/v1/admin/users" \
    -H "apikey: ${SERVICE_ROLE}" \
    -H "Authorization: Bearer ${SERVICE_ROLE}" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"email_confirm\":true}" >/dev/null

  LOGIN=$(curl -sf -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"device_id\":\"$DEVICE_ID\",\"platform\":\"ios\"}")
  printf '%s' "$LOGIN"
}

echo ">>> 1. health"
check_server
curl -sf "$BASE_URL/health" | grep -q ok && echo "OK"

echo ">>> 2. login ($EMAIL)"
LOGIN=$(ensure_login)
TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN")
REFRESH=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['refresh_token'])" <<<"$LOGIN")
SESSION=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$LOGIN")
if [[ -z "$REFRESH" ]]; then
  echo "失败: login 未返回 refresh_token" >&2
  exit 1
fi
echo "OK session_id=$SESSION refresh_token=***"

echo ">>> 3. refresh token"
REFRESH_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/user/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH\"}")
TOKEN2=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$REFRESH_RESP")
REFRESH2=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['refresh_token'])" <<<"$REFRESH_RESP")
if [[ -z "$TOKEN2" || -z "$REFRESH2" ]]; then
  echo "失败: refresh 未返回新 token" >&2
  echo "$REFRESH_RESP" >&2
  exit 1
fi
echo "OK new token received"

echo ">>> 4. transactions with refreshed token (expect 200)"
HTTP=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN2" \
  -H "X-Session-ID: $SESSION" \
  -H "X-Device-ID: $DEVICE_ID" \
  "$BASE_URL/api/v1/transactions?page=1&size=1")
if [[ "$HTTP" != "200" ]]; then
  echo "失败: refresh 后业务 API 期望 200，实际 $HTTP" >&2
  exit 1
fi
echo "OK ($HTTP)"

echo ">>> 5. logout"
LOGOUT_HTTP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/user/logout" \
  -H "Authorization: Bearer $TOKEN2" \
  -H "X-Session-ID: $SESSION" \
  -H "X-Device-ID: $DEVICE_ID")
if [[ "$LOGOUT_HTTP" != "200" ]]; then
  echo "失败: logout 期望 200，实际 $LOGOUT_HTTP" >&2
  exit 1
fi
echo "OK ($LOGOUT_HTTP)"

echo ">>> 6. transactions after logout (expect 401)"
HTTP_AFTER=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN2" \
  -H "X-Session-ID: $SESSION" \
  -H "X-Device-ID: $DEVICE_ID" \
  "$BASE_URL/api/v1/transactions?page=1&size=1")
if [[ "$HTTP_AFTER" != "401" ]]; then
  echo "失败: logout 后期望 401，实际 $HTTP_AFTER" >&2
  exit 1
fi
echo "OK ($HTTP_AFTER)"

echo ">>> refresh/logout 联调通过"
