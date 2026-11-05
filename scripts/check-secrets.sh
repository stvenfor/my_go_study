#!/usr/bin/env bash
# 推送前检查：阻止真实密钥与占位默认值进入 Git 跟踪文件。
#
# 三档判定：
#   ❌ 硬失败 —— service_role 真值、secret 类变量的高熵真值、非 lab 文件里的占位默认值 / 长度 <16
#   ⚠️ 提示   —— 有意入库的 lab 文件（.env）里的占位弱值：本地可用，部署必改
#   CHECK_SECRETS_STRICT=1 把「提示」也升级为失败（部署前 / CI 用）
#
# 用法：./scripts/check-secrets.sh [额外文件...]   （CI 里检查样例或临时 env 片段）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

FAILED=0
WARN_COUNT=0

# service_role 一类的真密钥特征（旧规则原样保留 = 回归基线）
readonly SERVICE_ROLE_RE='sb_secret_[A-Za-z0-9_-]+|SUPABASE_SERVICE_ROLE_KEY=sb_'
# 变量名像密钥的才做强度判定（避免 DATABASE_HOST 之类误报）
readonly SECRET_KEY_RE='(SECRET|PASSWORD|PASSWD|TOKEN|API_KEY|APP_SECRET|MASTER_SECRET|PRIVATE_KEY|CLIENT_SECRET|SERVICE_ROLE)'
# 设计上就要公开的标识位（anon key / app id），不参与弱值与高熵判定
readonly PUBLIC_ID_RE='^(SUPABASE_ANON_KEY|SUPABASE_KEY|WECHAT_APP_ID|WECHAT_MCH_ID|ALIPAY_APP_ID|HUAWEI_CLIENT_ID|HUAWEI_IAP_APP_ID|HUAWEI_IAP_KEY_ID|HUAWEI_IAP_ISSUER_ID|JPUSH_APP_KEY|RONGCLOUD_APP_KEY)$'
# 脚手架/教程默认值
WEAK_VALUES=(change-me change-me-in-dev change-me-in-lan change-me-in-production your-secret jwt-secret secret postgres password admin root 123456 dev-test-phone-secret)
# 有意入库的联调文件（历史 commit afbc29f）：弱值只提示，不拦推送
LAB_FILES=(".env")
readonly MIN_SECRET_LEN=16
readonly HIGH_ENTROPY_LEN=20

is_lab_file() {
  local lab
  for lab in "${LAB_FILES[@]}"; do
    [[ "$1" == "$lab" ]] && return 0
  done
  return 1
}

is_env_style() {
  case "$1" in
  .env* | *.env | *.env.*) return 0 ;;
  *) return 1 ;;
  esac
}

# *.example 是模板，占位值正是它该有的内容 —— 只查 service_role
is_template_file() {
  case "$1" in
  *.example) return 0 ;;
  *) return 1 ;;
  esac
}

is_weak_value() {
  local lowered
  lowered="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
  local w
  for w in "${WEAK_VALUES[@]}"; do
    [[ "$lowered" == "$w" ]] && return 0
  done
  return 1
}

# 命中 ≥3 类字符（大小写/数字/符号）且够长 —— 视为「像真密钥」
high_entropy() {
  local v="$1" classes=0
  [[ "${#v}" -lt $HIGH_ENTROPY_LEN ]] && return 1
  [[ "$v" =~ [a-z] ]] && classes=$((classes + 1))
  [[ "$v" =~ [A-Z] ]] && classes=$((classes + 1))
  [[ "$v" =~ [0-9] ]] && classes=$((classes + 1))
  [[ "$v" =~ [^A-Za-z0-9] ]] && classes=$((classes + 1))
  [[ "$classes" -ge 3 ]]
}

fail() {
  echo "❌ $1"
  FAILED=1
}

warn() {
  echo "⚠️  $1"
  WARN_COUNT=$((WARN_COUNT + 1))
}

# 第一层：任何文件都只查 service_role 真值（沿用旧能力）
check_service_role() {
  local file="$1"
  if grep -vE '^\s*#' "$file" | grep -qE "$SERVICE_ROLE_RE"; then
    fail "$file 含 Supabase service_role 密钥，GitHub 会拒绝推送（请移到 .env.local）"
  fi
}

# 第二层：env 风格文件里的密钥变量做占位 / 长度 / 熵判定
check_env_values() {
  local file="$1"
  is_env_style "$file" || return 0
  is_template_file "$file" && return 0

  local lab=0
  is_lab_file "$file" && lab=1

  local line key value
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" != *=* ]] && continue
    key="${line%%=*}"
    value="${line#*=}"
    key="${key//[[:space:]]/}"
    value="${value%$'\r'}"
    value="${value%\"}"; value="${value#\"}"
    value="${value%\'}"; value="${value#\'}"

    [[ "$key" =~ $SECRET_KEY_RE ]] || continue
    [[ "$key" =~ $PUBLIC_ID_RE ]] && continue
    [[ -z "$value" ]] && continue

    if [[ $lab -eq 1 ]]; then
      is_weak_value "$value" && warn "$file:$key 仍是占位默认值（lab 有意入库；部署前必改，见 docs/startup-guide.md）"
      continue
    fi
    if is_weak_value "$value"; then
      fail "$file:$key 使用占位默认值 —— 本地沿用 .env.example，部署必须注入随机串"
    elif [[ "$value" =~ [[:space:]] ]]; then
      fail "$file:$key 含空白字符（多为复制粘贴或引号写法出错）"
    elif high_entropy "$value"; then
      fail "$file:$key 疑似真实密钥（高熵，长度 ${#value}）—— 移出可提交文件，改放 .env.local 或 OS env"
    elif [[ "${#value}" -lt $MIN_SECRET_LEN ]]; then
      fail "$file:$key 长度 ${#value} < ${MIN_SECRET_LEN}，过短"
    fi
  done <"$file"
}

scan_file() {
  local file="$1"
  [[ -f "$file" ]] || return 0
  check_service_role "$file"
  check_env_values "$file"
}

check_tracked() {
  local file="$1"
  git check-ignore -q "$file" 2>/dev/null && return 0
  git ls-files --error-unmatch "$file" >/dev/null 2>&1 || return 0
  scan_file "$file"
}

# 已入库的 env 类文件
for f in configs/supabase.env configs/supabase.env.example .env .env.example configs/config.dev.yaml configs/config.yaml configs/config.prod.yaml; do
  [[ -f "$f" ]] && check_tracked "$f"
done

# 暂存区中的新增/修改
while IFS= read -r path; do
  case "$path" in
  *.env | *.env.* | configs/*.yaml) scan_file "$path" ;;
  esac
done < <(git diff --cached --name-only 2>/dev/null || true)

# 显式传入的文件
for path in "$@"; do
  scan_file "$path"
done

if [[ "$FAILED" -ne 0 ]]; then
  echo ""
  echo "运行 ./scripts/check-secrets.sh 通过后再推送。"
  echo "本地密钥请写入 .env.local（见 .env.local.example）。"
  exit 1
fi

if [[ "$WARN_COUNT" -ne 0 && "${CHECK_SECRETS_STRICT:-0}" == "1" ]]; then
  echo ""
  echo "❌ CHECK_SECRETS_STRICT=1：$WARN_COUNT 条占位弱值在部署口径下不允许。"
  exit 1
fi

if [[ "$WARN_COUNT" -ne 0 ]]; then
  echo "✅ 密钥检查通过（无真实密钥入库；另有 $WARN_COUNT 条 lab 弱值提示，部署前需改掉）"
else
  echo "✅ 密钥检查通过（入库文件未含 service_role 与占位密钥）"
fi
