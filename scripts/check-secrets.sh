#!/usr/bin/env bash
# 推送前检查：阻止 service_role 等密钥进入 Git 跟踪文件。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

FAILED=0

check_file_content() {
  local file="$1"
  # 仅检查非注释行的赋值，避免说明文字误报
  if grep -vE '^\s*#' "$file" | grep -qE 'sb_secret_[A-Za-z0-9_-]+|SUPABASE_SERVICE_ROLE_KEY=sb_'; then
    echo "❌ $file 含 Supabase service_role 密钥，GitHub 会拒绝推送"
    echo "   请将 SUPABASE_SERVICE_ROLE_KEY 移到 .env.local（不入库）"
    FAILED=1
  fi
}

check_tracked() {
  local file="$1"
  if git check-ignore -q "$file" 2>/dev/null; then
    return 0
  fi
  if ! git ls-files --error-unmatch "$file" >/dev/null 2>&1; then
    return 0
  fi
  [[ -f "$file" ]] && check_file_content "$file"
}

# 已入库的 env 类文件
for f in .env .env.example configs/config.dev.yaml configs/config.yaml configs/config.prod.yaml; do
  [[ -f "$f" ]] && check_tracked "$f"
done

# 暂存区中的新增/修改
while IFS= read -r -d '' path; do
  case "$path" in
    *.env|*.env.*|configs/*.yaml)
      if grep -qE "$PATTERN" "$path" 2>/dev/null; then
        echo "❌ 暂存区 $path 含 service_role 密钥"
        FAILED=1
      fi
      ;;
  esac
done < <(git diff --cached --name-only -z 2>/dev/null || true)

if [[ "$FAILED" -ne 0 ]]; then
  echo ""
  echo "运行 ./scripts/check-secrets.sh 通过后再推送。"
  echo "本地密钥请写入 .env.local（见 .env.local.example）。"
  exit 1
fi

echo "✅ 密钥检查通过（入库文件未含 service_role）"
