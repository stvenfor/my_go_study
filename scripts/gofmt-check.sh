#!/usr/bin/env bash
# gofmt-check.sh — harden-h3「gofmt -l」门禁（不新增欠账口径）
#
# 为什么不是裸 `gofmt -l $(git ls-files '*.go')`：
#   HEAD 上有 10 个 tracked .go 本就未 gofmt（见同目录 gofmt-baseline.txt），
#   业务 .go 属本 Slice 黑名单不能改。裸门禁会恒红，CI 就没法在 PR 上真跑、
#   也没法观察「故意塞一个未 gofmt 文件」被拦。故用 ratchet：只拦新增欠账。
#
# 扫描范围：tracked .go + 未忽略的 untracked .go（干净 CI 检出 = 仅 tracked；
#   本机多扫 untracked 是为了让临时塞进来的未格式化文件也能被拦）。
# 依赖：git + gofmt（随 Go 工具链自带，无新增依赖）。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
BASELINE="$(cd "$(dirname "$0")" && pwd)/gofmt-baseline.txt"

files=()
while IFS= read -r f; do
  [ -n "$f" ] && files+=("$f")
done < <({ git ls-files '*.go'; git ls-files --others --exclude-standard -- '*.go'; } | sort -u)

if [ "${#files[@]}" -eq 0 ]; then
  echo "OK: 没有 .go 文件，gofmt 门禁通过"
  exit 0
fi

unformatted="$(gofmt -l "${files[@]}")"

# 去掉 baseline 里登记的既有欠账（精确整行）与空行，剩下的即「新增欠账」
new=""
if [ -n "$unformatted" ]; then
  new="$(printf '%s\n' "$unformatted" | grep -vxF -f "$BASELINE" | grep -v '^[[:space:]]*$' || true)"
fi

if [ -n "$new" ]; then
  echo "FAIL: 以下 .go 未 gofmt，且不在 baseline 欠账清单:" >&2
  printf '  %s\n' "$new" >&2
  echo "修法: gofmt -w <文件>；确属历史欠账再登记进 $BASELINE" >&2
  exit 1
fi

n_base="$(grep -vcE '^[[:space:]]*(#|$)' "$BASELINE")"
echo "OK: gofmt 门禁通过（既有 ${n_base} 个欠账文件已登记，未新增；扫描 ${#files[@]} 个 .go）"
