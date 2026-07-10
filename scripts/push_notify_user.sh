#!/usr/bin/env bash
# 向指定 userId 推送 sys.notify（开发环境 push_async=false 时同步直投 WS）。
#
# 用法:
#   ./scripts/push_notify_user.sh [userId] [email] [title] [body]
#   make push-notify-user USER_ID=... EMAIL=... TITLE='...' BODY='...'
#
# 可选环境变量（跳过临时建号，加快推送）:
#   PUSH_OPERATOR_TOKEN   已登录 BFF 的 Bearer token
#   PUSH_OPERATOR_SESSION X-Session-ID
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

TARGET_USER_ID="${1:-3704f664-5f2c-4ea9-acdf-f244256dc935}"
TARGET_EMAIL="${2:-454655062@qq.com}"
TITLE="${3:-即时通知}"
BODY="${4:-WebSocket 推送测试 $(date '+%H:%M:%S')}"
BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
CURL_CONNECT="${CURL_CONNECT:-15}"
CURL_MAX="${CURL_MAX:-60}"

curl_json() {
  local label="$1"
  shift
  local out http rc
  set +e
  out=$(curl -sS --connect-timeout "$CURL_CONNECT" --max-time "$CURL_MAX" -w "\n%{http_code}" "$@")
  rc=$?
  set -e
  if [[ "$rc" -ne 0 ]]; then
    echo "错误: $label 失败 (curl 退出码 $rc，28=超时)" >&2
    echo "  请确认 API/Supabase 可达；可先 make run，或 export CURL_MAX=120" >&2
    [[ -n "${out:-}" ]] && echo "$out" >&2
    exit "$rc"
  fi
  http="${out##*$'\n'}"
  out="${out%$'\n'*}"
  if [[ "$http" -lt 200 || "$http" -ge 300 ]]; then
    echo "错误: $label HTTP $http" >&2
    echo "$out" >&2
    exit 1
  fi
  printf '%s' "$out"
}

echo ">>> 检查 API $BASE_URL/health"
if ! curl -sf --connect-timeout 5 --max-time 10 "$BASE_URL/health" >/dev/null; then
  echo "错误: API 未启动或不可达，请先在 my_go_study/ 执行: make run" >&2
  exit 1
fi

if [[ -z "${SUPABASE_URL:-}" || -z "${SUPABASE_SERVICE_ROLE_KEY:-}" ]]; then
  echo "错误: 缺少 SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY（configs/supabase.env + .env.local）" >&2
  exit 1
fi

TOKEN="${PUSH_OPERATOR_TOKEN:-}"
SESSION_ID="${PUSH_OPERATOR_SESSION:-}"

if [[ -z "$TOKEN" || -z "$SESSION_ID" ]]; then
  OP_EMAIL="push_op_$(date +%s)@gmail.com"
  OP_PASS="TestPass123!"
  echo ">>> 创建临时操作员 ${OP_EMAIL} (可设 PUSH_OPERATOR_TOKEN 跳过)"
  curl_json "Supabase 创建用户" \
    -X POST "${SUPABASE_URL}/auth/v1/admin/users" \
    -H "apikey: ${SUPABASE_SERVICE_ROLE_KEY}" \
    -H "Authorization: Bearer ${SUPABASE_SERVICE_ROLE_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${OP_EMAIL}\",\"password\":\"${OP_PASS}\",\"email_confirm\":true}" >/dev/null

  echo ">>> BFF 登录获取推送凭证"
  LOGIN_RESP=$(curl_json "BFF login" \
    -X POST "$BASE_URL/api/v1/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${OP_EMAIL}\",\"password\":\"${OP_PASS}\",\"device_id\":\"push-script\",\"platform\":\"ios\"}")

  TOKEN=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])" <<<"$LOGIN_RESP")
  SESSION_ID=$(python3 -c "import json,sys; print(json.load(sys.stdin)['data']['session_id'])" <<<"$LOGIN_RESP")
fi

PUSH_BODY=$(TARGET_USER_ID="$TARGET_USER_ID" TITLE="$TITLE" BODY="$BODY" TARGET_EMAIL="$TARGET_EMAIL" python3 <<'PY'
import json, os
print(json.dumps({
  "userId": os.environ["TARGET_USER_ID"],
  "title": os.environ["TITLE"],
  "body": os.environ["BODY"],
  "extra": {
    "category": "manual",
    "source": "push_notify_user.sh",
    "metadata": {"targetEmail": os.environ["TARGET_EMAIL"]},
  },
}, ensure_ascii=False))
)

echo ">>> POST /api/v1/realtime/push -> $TARGET_EMAIL ($TARGET_USER_ID)"
PUSH_RESP=$(curl_json "realtime push" \
  -X POST "$BASE_URL/api/v1/realtime/push" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Session-ID: $SESSION_ID" \
  -H "X-Device-ID: push-script" \
  -H "Content-Type: application/json" \
  -d "$PUSH_BODY")

python3 -c "
import json,sys
r=json.load(sys.stdin)
print('queued:', r.get('queued'))
print('delivered:', r.get('delivered'))
print('taskId:', r.get('taskId'))
env=r.get('envelope') or {}
print('envelope.id:', env.get('id'))
print('envelope.seq:', env.get('seq'))
payload=(env.get('payload') or {})
print('title:', payload.get('title'))
print('body:', payload.get('body'))
print('notifyId:', payload.get('notifyId'))
d=r.get('delivered')
if d == 0:
    print('WARN: delivered=0，目标用户当前无已订阅 sys.notify 的 WS 连接 (重开 App 或 sync 补拉)')
elif d and d > 0:
    print('OK: 已实时送达', d, '条 WS 连接')
" <<<"$PUSH_RESP"
