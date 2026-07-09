#!/usr/bin/env bash
# 测试环境固定手机号 OTP 联调：13400000000 + 123456 登录并访问受保护接口。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
TEST_PHONE="${TEST_PHONE:-13400000000}"
TEST_OTP="${TEST_OTP:-123456}"
DEVICE_ID="${TEST_DEVICE_ID:-phone-otp-$(date +%s)}"
PLATFORM="${TEST_PLATFORM:-ios}"

check_server() {
  if ! curl -sf --connect-timeout 3 "$BASE_URL/health" >/dev/null; then
    echo "错误: $BASE_URL/health 不可达"
    echo "  请先在另一终端执行: cd my_go_study && make run"
    exit 1
  fi
}

echo ">>> 1. health"
check_server
curl -sf "$BASE_URL/health" | grep -q ok && echo "OK"

echo ">>> 2. send OTP ($TEST_PHONE)"
SEND_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/user/phone/otp/send" \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"$TEST_PHONE\"}")
python3 -c "import json,sys; d=json.load(sys.stdin); assert d.get('code')==0, d" <<<"$SEND_RESP"
echo "OK"

echo ">>> 3. verify OTP"
VERIFY_RESP=$(curl -sf -X POST "$BASE_URL/api/v1/user/phone/otp/verify" \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"$TEST_PHONE\",\"otp\":\"$TEST_OTP\",\"device_id\":\"$DEVICE_ID\",\"platform\":\"$PLATFORM\"}")
TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$VERIFY_RESP")
SESSION_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$VERIFY_RESP")
echo "session_id=$SESSION_ID"

echo ">>> 4. transactions with session headers (expect 200)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Session-ID: $SESSION_ID" \
  -H "X-Device-ID: $DEVICE_ID" \
  "$BASE_URL/api/v1/transactions?page=1&size=1")
if [[ "$HTTP_CODE" != "200" ]]; then
  echo "失败: 期望 200，实际 $HTTP_CODE"
  exit 1
fi
echo "OK ($HTTP_CODE)"

echo ">>> 5. wrong OTP (expect error)"
WRONG_BODY=$(curl -s -X POST "$BASE_URL/api/v1/user/phone/otp/verify" \
  -H "Content-Type: application/json" \
  -d "{\"phone\":\"$TEST_PHONE\",\"otp\":\"000000\",\"device_id\":\"$DEVICE_ID\",\"platform\":\"$PLATFORM\"}")
if ! echo "$WRONG_BODY" | grep -q "验证码错误"; then
  echo "失败: 错误 OTP 应返回验证码错误，实际: $WRONG_BODY"
  exit 1
fi
echo "OK"

echo "全部通过：测试手机号 OTP 登录"
