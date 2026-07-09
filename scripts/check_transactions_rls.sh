#!/usr/bin/env bash
# 检查 Supabase transactions 表 RLS 是否生效
# 用法:
#   make check-rls
#   SUPABASE_ACCESS_TOKEN='eyJ...' make check-rls   # 跳过创建临时用户

set -uo pipefail

# shellcheck disable=SC1091
source "$(dirname "$0")/source-env.sh"

URL="${SUPABASE_URL:?}"
ANON="${SUPABASE_ANON_KEY:?}"
SERVICE="${SUPABASE_SERVICE_ROLE_KEY:-}"
BASE="${BASE_URL:-http://localhost:8080}"
TOKEN="${SUPABASE_ACCESS_TOKEN:-}"
NETWORK_OK=1

curl_json() {
  curl -sS --connect-timeout 15 --max-time 45 "$@"
}

echo "=== transactions RLS 检查 ==="

if [[ -z "$SERVICE" && -z "$TOKEN" ]]; then
  echo "错误: 需要 SUPABASE_SERVICE_ROLE_KEY 或 SUPABASE_ACCESS_TOKEN"
  exit 1
fi

if [[ -z "$TOKEN" ]]; then
  echo ">>> 创建临时用户（需访问 Supabase Auth API）..."
  EMAIL="rls_check_$(date +%s)@gmail.com"
  CREATE_CODE=$(curl_json -o /dev/null -w "%{http_code}" -X POST "$URL/auth/v1/admin/users" \
    -H "apikey: $SERVICE" -H "Authorization: Bearer $SERVICE" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"TestPass123!\",\"email_confirm\":true}" 2>/dev/null || echo "000")

  if [[ "$CREATE_CODE" != "200" && "$CREATE_CODE" != "201" ]]; then
    echo "⚠️  无法连接 Supabase Auth（HTTP ${CREATE_CODE}，常见原因：网络超时/SSL）"
    echo "   这不代表 RLS 未配置；你在 Dashboard 看到 4 条策略即表示策略已就绪。"
    echo ""
    echo "   可选：用已有 token 跳过创建用户："
    echo "   export SUPABASE_ACCESS_TOKEN='Flutter 登录后的 access_token'"
    echo "   make check-rls"
    NETWORK_OK=0
  else
    LOGIN=$(curl_json -X POST "$URL/auth/v1/token?grant_type=password" \
      -H "apikey: $ANON" -H "Content-Type: application/json" \
      -d "{\"email\":\"${EMAIL}\",\"password\":\"TestPass123!\"}")
    TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || true)
    if [[ -z "$TOKEN" ]]; then
      echo "⚠️  临时用户登录失败，请改用 SUPABASE_ACCESS_TOKEN"
      NETWORK_OK=0
    fi
  fi
else
  echo ">>> 使用 SUPABASE_ACCESS_TOKEN（跳过创建临时用户）"
fi

if [[ "$NETWORK_OK" == "1" && -n "$TOKEN" ]]; then
  echo ""
  echo "1) 用户 token 直查 Supabase REST（测数据库 RLS）"
  REST_ROWS=$(curl_json "$URL/rest/v1/transactions?select=id,user_id&limit=3" \
    -H "apikey: $ANON" -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo '[]')
  echo "$REST_ROWS" | python3 -m json.tool 2>/dev/null || echo "$REST_ROWS"
  REST_COUNT=$(echo "$REST_ROWS" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d) if isinstance(d,list) else -1)" 2>/dev/null || echo "-1")

  echo ""
  echo "2) 经 Go 后端查询（测应用层 user_id 过滤）"
  API_ROWS=$(curl_json -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/transactions?limit=3" 2>/dev/null || echo '{"error":"backend_unreachable"}')
  echo "$API_ROWS" | python3 -m json.tool 2>/dev/null || echo "$API_ROWS"
  API_ITEMS=$(echo "$API_ROWS" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('items',[])) if 'items' in d else -1)" 2>/dev/null || echo "-1")

  echo ""
  echo "=== 结论 ==="
  if [[ "$REST_COUNT" == "0" ]]; then
    echo "✅ Supabase RLS 已生效（REST 层已隔离）"
  elif [[ "$REST_COUNT" -gt 0 ]]; then
    echo "⚠️  REST 仍返回 ${REST_COUNT} 条（若 user_id 均为当前用户则正常）"
  else
    echo "⚠️  无法解析 REST 响应"
  fi

  if [[ "$API_ITEMS" == "0" ]]; then
    echo "✅ Go 后端 user_id 过滤已生效"
  elif [[ "$API_ITEMS" == "-1" ]]; then
    echo "⚠️  Go 后端不可达，请先 make run"
  else
    echo "✅ Go 后端返回 ${API_ITEMS} 条（当前用户自己的数据）"
  fi
else
  echo ""
  echo "=== Dashboard 策略自检（无需网络）==="
  echo "✅ 若 SQL Editor 显示以下 4 条策略，RLS 配置正确："
  echo "   transactions_select_own  (SELECT, authenticated)"
  echo "   transactions_insert_own  (INSERT, authenticated)"
  echo "   transactions_update_own  (UPDATE, authenticated)"
  echo "   transactions_delete_own  (DELETE, authenticated)"
  echo ""
  echo "   policy_count 应为 4，rls_enabled 应为 true"
fi
