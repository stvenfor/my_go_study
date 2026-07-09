#!/usr/bin/env bash
# 单设备登录联调：设备 B 登录后，设备 A 的旧 session 应返回 401。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
EMAIL="${TEST_EMAIL:-demo@example.com}"
PASSWORD="${TEST_PASSWORD:-123456}"
DEVICE_A="${TEST_DEVICE_A:-device-a-$(date +%s)}"
DEVICE_B="${TEST_DEVICE_B:-device-b-$(date +%s)}"

check_server() {
  if ! curl -sf --connect-timeout 3 "$BASE_URL/health" >/dev/null; then
    echo "错误: $BASE_URL/health 不可达"
    echo "  请先在另一终端执行: cd my_go_study && make run"
    exit 1
  fi
}

login_device() {
  local device_id="$1"
  local platform="$2"
  curl -sf -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"device_id\":\"$device_id\",\"platform\":\"$platform\"}"
}

echo ">>> 1. health"
check_server
curl -sf "$BASE_URL/health" | grep -q ok && echo "OK"

echo ">>> 2. device A login ($DEVICE_A)"
LOGIN_A=$(login_device "$DEVICE_A" "ios")
TOKEN_A=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN_A")
SESSION_A=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$LOGIN_A")
echo "device A session_id=$SESSION_A"

echo ">>> 3. device A transactions (expect 200)"
HTTP_A=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "X-Session-ID: $SESSION_A" \
  -H "X-Device-ID: $DEVICE_A" \
  "$BASE_URL/api/v1/transactions?page=1&size=1")
if [[ "$HTTP_A" != "200" ]]; then
  echo "失败: device A 首次请求期望 200，实际 $HTTP_A"
  exit 1
fi
echo "OK ($HTTP_A)"

echo ">>> 4. device B login ($DEVICE_B)"
LOGIN_B=$(login_device "$DEVICE_B" "android")
TOKEN_B=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN_B")
SESSION_B=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$LOGIN_B")
echo "device B session_id=$SESSION_B"

echo ">>> 5. device B transactions (expect 200)"
HTTP_B=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN_B" \
  -H "X-Session-ID: $SESSION_B" \
  -H "X-Device-ID: $DEVICE_B" \
  "$BASE_URL/api/v1/transactions?page=1&size=1")
if [[ "$HTTP_B" != "200" ]]; then
  echo "失败: device B 请求期望 200，实际 $HTTP_B"
  exit 1
fi
echo "OK ($HTTP_B)"

echo ">>> 6. device A transactions again (expect 401)"
BODY_A=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
  -H "Authorization: Bearer $TOKEN_A" \
  -H "X-Session-ID: $SESSION_A" \
  -H "X-Device-ID: $DEVICE_A" \
  "$BASE_URL/api/v1/transactions?page=1&size=1")
HTTP_A2=$(echo "$BODY_A" | tail -n1 | sed 's/HTTP_CODE://')
MSG_A=$(echo "$BODY_A" | head -n -1)
if [[ "$HTTP_A2" != "401" ]]; then
  echo "失败: device A 二次请求期望 401，实际 $HTTP_A2"
  echo "$MSG_A"
  exit 1
fi
if ! echo "$MSG_A" | grep -q "其他设备登录"; then
  echo "失败: 401 响应应包含「其他设备登录」"
  echo "$MSG_A"
  exit 1
fi
echo "OK (401 + 其他设备登录)"

echo ">>> 单设备登录联调通过"
