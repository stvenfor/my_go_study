#!/usr/bin/env bash
# check-brief.sh — harden-h3：Slice 合同机检（make agent-pre → check-brief）
#
# 依据 .harness/rules/slice-contract.md：
#   * 每个 plans/slices/<id>.md 的 ## Slice Brief 区域内必须「含」六节字段头：
#     本轮 ONLY / 不做 / 验收 / 文件白名单 / 验证命令 / 证据。
#   * 「子列表项必须缩进」：## Slice Brief 区域内的子列表项必须缩进。
#
# 口径说明（为什么这样实现才「不误报 / 不假放行」）：
#   - 六节必须作为**字段头**出现在 ## Slice Brief 区域内：
#     `^- (**)?<节名>(**)?[^:：]*[:：]`（允许 `- **本轮 ONLY**（4 项）:` 这类加粗变体）。
#     不再用全文 grep -q：正文 / ## Context Card 里出现关键词不算，否则「整节漏写」
#     也能蒙混过关（harden-h10 就曾只在正文出现「证据」而 Brief 无该节）。
#   - 缩进只查 ## Slice Brief 区域：该区域内顶格 `- ` 行必须是字段头（含 : 或 ：）；
#     否则即「未缩进的子列表项」。## 决策门 / ## Context Card 等其它区域的顶格
#     列表项不在此规则约束内，故不误报。
#   - 一个 slice 都没有时必须 FAIL，不能打印 OK（防空跑）。
#
# 用法：
#   ./scripts/check-brief.sh                  校验全部 tracked 的 plans/slices/*.md
#   ./scripts/check-brief.sh <file> [file]    校验指定文件
#   ./scripts/check-brief.sh --self-test      造 4 个临时样例自证「能拦也能放」
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

SECTIONS=("本轮 ONLY" "不做" "验收" "文件白名单" "验证命令" "证据")

check_one() {
  local f="$1" s bad=0 offenders ln brief
  [ -f "$f" ] || { echo "  FAIL $f: 文件不存在"; return 1; }

  # 只取 ## Slice Brief 区域（到下一个 ## 标题为止）。
  brief="$(awk '
    /^## Slice Brief/ { inb=1; next }
    inb && /^## /     { exit }
    inb               { print }
  ' "$f")"

  # 六节：必须是 Slice Brief 区内的字段头，不允许只出现在正文 / Context Card。
  for s in "${SECTIONS[@]}"; do
    if ! grep -qE -- "^- (\*\*)?${s}(\*\*)?[^:：]*[:：]" <<< "$brief"; then
      echo "  FAIL $f: Slice Brief 缺六节字段头「${s}」"
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
  local ok miss unind bodyonly rc=0
  ok="$(mktemp)"; miss="$(mktemp)"; unind="$(mktemp)"; bodyonly="$(mktemp)"
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
  # 反例：六节词只出现在正文 / Context Card，Slice Brief 内无任何字段头。
  cat > "$bodyonly" <<'EOF'
正文枚举了 本轮 ONLY / 不做 / 验收 / 文件白名单 / 验证命令 / 证据 六个词。

## Slice Brief
这里只有一段说明文字，没有任何字段头。

## Context Card
- 已完成: 本轮 ONLY、不做、验收、文件白名单、验证命令、证据 都在别处提过
EOF
  echo "[self-test] 合法样例应放行"
  if check_one "$ok" >/dev/null; then echo "  PASS 放行合法"; else echo "  FAIL 误杀合法"; rc=1; fi
  echo "[self-test] 缺一节样例应拦下"
  if check_one "$miss" >/dev/null; then echo "  FAIL 没拦住缺节"; rc=1; else echo "  PASS 拦下缺节"; fi
  echo "[self-test] 子列表未缩进样例应拦下"
  if check_one "$unind" >/dev/null; then echo "  FAIL 没拦住未缩进"; rc=1; else echo "  PASS 拦下未缩进"; fi
  echo "[self-test] 六节词只在正文、Brief 无字段头应拦下"
  if check_one "$bodyonly" >/dev/null; then echo "  FAIL 没拦住正文假放行"; rc=1; else echo "  PASS 拦下正文假放行"; fi
  rm -f "$ok" "$miss" "$unind" "$bodyonly"
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
if [ "${#files[@]}" -eq 0 ]; then
  echo "FAIL: 没有找到 plans/slices/*.md，check-brief 无法校验（不允许空跑）" >&2
  exit 1
fi
fail=0
for f in "${files[@]}"; do
  check_one "$f" || fail=1
done
if [ "$fail" -ne 0 ]; then
  echo "FAIL: check-brief 未通过" >&2
  exit 1
fi
echo "OK: check-brief 通过（${#files[@]} 个 slice 六节字段头齐全、子列表缩进）"
