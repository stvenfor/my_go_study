#!/usr/bin/env bash
# check-brief.sh — harden-h3：Slice 合同机检（make agent-pre → check-brief）
#
# 依据 .harness/rules/slice-contract.md：
#   * 每个 plans/slices/<id>.md 必须「含」六节：本轮 ONLY / 不做 / 验收 /
#     文件白名单 / 验证命令 / 证据。
#   * 「子列表项必须缩进」：## Slice Brief 区域内的子列表项必须缩进。
#
# 口径说明（为什么这样实现才「不误报」）：
#   - 六节用「字面存在」判定，而非强求 `- 本轮 ONLY:` 这样的字段头。原因是人批的
#     Brief 存在变体（例如 harden-h3 把「本轮 ONLY / 不做 / …」作为引号内的枚举
#     写在验收子项里），强解析会把已批 Brief 误杀。机检定位是拦「整节漏写」。
#   - 缩进只查 ## Slice Brief 区域：该区域内顶格 `- ` 行必须是字段头（- 名字:）；
#     否则即「未缩进的子列表项」。## 决策门 / ## Context Card 等其它区域的顶格
#     列表项不在此规则约束内，故不误报。
#
# 用法：
#   ./scripts/check-brief.sh                  校验全部 tracked 的 plans/slices/*.md
#   ./scripts/check-brief.sh <file> [file]    校验指定文件
#   ./scripts/check-brief.sh --self-test      造 3 个临时样例自证「能拦也能放」
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

SECTIONS=("本轮 ONLY" "不做" "验收" "文件白名单" "验证命令" "证据")

check_one() {
  local f="$1" s bad=0 offenders ln
  [ -f "$f" ] || { echo "  FAIL $f: 文件不存在"; return 1; }

  # 六节：字面存在
  for s in "${SECTIONS[@]}"; do
    if ! grep -q -- "${s}" "$f"; then
      # ${s} 必须花括号定界：bash 3.2 + set -u 下 "$s」" 会把全角括号的字节
      # 吞进变量名，误报 unbound variable。
      echo "  FAIL $f: 缺六节之一「${s}」"
      bad=1
    fi
  done

  # 子列表缩进：## Slice Brief 区域内，顶格 "- " 行必须是字段头（含 : 或 ：）
  offenders="$(awk '
    /^## Slice Brief/ { inb=1; next }
    inb && /^## /     { inb=0 }
    inb && /^- / {
      if ($0 !~ /^- [^:：]{1,45}[:：]/) print NR ":" $0
    }
  ' "$f")"
  if [ -n "$offenders" ]; then
    while IFS= read -r ln; do
      [ -n "$ln" ] || continue
      echo "  FAIL $f: 子列表未缩进（Slice Brief 区顶格列表项）-> $ln"
    done <<< "$offenders"
    bad=1
  fi

  [ "$bad" -eq 0 ] && echo "  OK    $f"
  return "$bad"
}

self_test() {
  local ok miss unind rc=0
  ok="$(mktemp)"; miss="$(mktemp)"; unind="$(mktemp)"
  cat > "$ok" <<'EOF'
## Slice Brief
- 本轮 ONLY: x
- 不做: y
- 验收:
  - [ ] a
- 文件白名单:
  - Makefile
- 验证命令:
  - make test
- 证据: docs/x.md

## Context Card
- 已完成:
EOF
  cat > "$miss" <<'EOF'
## Slice Brief
- 本轮 ONLY: x
- 不做: y
- 验收:
  - [ ] a
- 文件白名单:
  - Makefile
- 验证命令:
  - make test

## Context Card
- 已完成:
EOF
  cat > "$unind" <<'EOF'
## Slice Brief
- 本轮 ONLY: x
- 不做: y
- 验收:
- [ ] a
- 文件白名单:
  - Makefile
- 验证命令:
  - make test
- 证据: docs/x.md

## Context Card
- 已完成:
EOF
  echo "[self-test] 合法样例应放行"
  if check_one "$ok" >/dev/null; then echo "  PASS 放行合法"; else echo "  FAIL 误杀合法"; rc=1; fi
  echo "[self-test] 缺一节样例应拦下"
  if check_one "$miss" >/dev/null; then echo "  FAIL 没拦住缺节"; rc=1; else echo "  PASS 拦下缺节"; fi
  echo "[self-test] 子列表未缩进样例应拦下"
  if check_one "$unind" >/dev/null; then echo "  FAIL 没拦住未缩进"; rc=1; else echo "  PASS 拦下未缩进"; fi
  rm -f "$ok" "$miss" "$unind"
  [ "$rc" -eq 0 ] && echo "[self-test] PASS" || echo "[self-test] FAIL"
  return "$rc"
}

if [ "${1:-}" = "--self-test" ]; then
  self_test
  exit $?
fi

files=()
if [ "$#" -gt 0 ]; then
  files=("$@")
else
  while IFS= read -r f; do files+=("$f"); done < <(git ls-files 'plans/slices/*.md')
fi

echo "[check-brief] 六节 = ${SECTIONS[*]}；共 ${#files[@]} 个 slice"
fail=0
for f in "${files[@]}"; do
  check_one "$f" || fail=1
done
if [ "$fail" -ne 0 ]; then
  echo "FAIL: check-brief 未通过" >&2
  exit 1
fi
echo "OK: check-brief 通过（${#files[@]} 个 slice 六节齐全、子列表缩进）"
