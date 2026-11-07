#!/usr/bin/env bash
# check-tracked-imports.sh — harden-h1「import 覆盖」机检
#
# 机检这一类缺陷：某个「仓库内包」被 import，但其目录里没有任何 tracked 文件
#   → 干净检出（git archive HEAD / CI checkout）里该包不存在 → go build 失败。
#   · 原 R1：app/internal/rag、core/authz、app/internal/txdb、app/internal/schemaboot（已随 3f0963e 入库）。
#   · 现行真阳性：thunder/logs/logs.go 被 thunder/{midd,req,upload} 三处 import，却因
#     .gitignore 的裸 `logs/` 规则被排除在跟踪之外 → `go build all` 报
#     "no required module provides package github.com/mszlu521/thunder/logs"。
#     详见 docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md。
#
# 口径：仓库内包 = go.work 的五个本地 module：app / common / core / model /
#       github.com/mszlu521/thunder。遍历所有 tracked .go 的 import，凡落到这五个
#       module 的，其目录必须有 tracked 的直接子级文件。
# 附带规则：git 跟踪文件不得 >1MB（防二进制误入库，Brief 验收第 4 条）。
#
# 设计：纯 git（ls-files / cat-file / check-ignore），不依赖 go.work 内容、go 工具链、
#       go mod tidy / go get 或网络，故可在干净检出 / 无外网 CI 里重复运行。
#
# 用法：
#   ./scripts/check-tracked-imports.sh              主检：输出缺失包/大文件列表，期望 0；有则 exit 1
#   ./scripts/check-tracked-imports.sh --self-test  自测：thunder/logs 回归锚点 + 映射/谓词单测，exit 0 = 通过
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

LIMIT=$((1024 * 1024))   # 1MB

# 仓库内 import 路径 → 仓库内目录；非仓库内包返回 1（不适用）
map_dir() {
  case "$1" in
    app|app/*|common|common/*|core|core/*|model|model/*)
      printf '%s' "$1" ;;
    github.com/mszlu521/thunder)
      printf 'thunder' ;;
    github.com/mszlu521/thunder/*)
      printf 'thunder/%s' "${1#github.com/mszlu521/thunder/}" ;;
    *)
      return 1 ;;
  esac
}

# 从所有 tracked .go 抽取 import 路径（单行 import 与 import(...) 块），去重排序
collect_imports() {
  git ls-files -z -- '*.go' | xargs -0 awk '
    FNR==1 { inb=0 }
    /^[[:space:]]*import[[:space:]]*\(/ { inb=1; next }
    inb && /^[[:space:]]*\)/            { inb=0; next }
    {
      line=$0
      sub(/\/\/.*$/, "", line)
      if (match(line, /"[^"]+"/)) {
        p = substr(line, RSTART+1, RLENGTH-2)
        if (inb==1 || line ~ /^[[:space:]]*import[[:space:]]/) print p
      }
    }
  ' | sort -u
}

# 核心谓词：$1=仓库内目录，$2=tracked 文件清单（换行分隔）。
# 目录中有「直接子级」的 tracked 文件 => 已覆盖(0)；否则未覆盖(1)。
# 同一逻辑供主检与 --self-test 共用，故自测不依赖仓库当前是否已修复。
covered() {
  local dir="$1" tracked="$2" f rest
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    case "$f" in
      "$dir"/*)
        rest="${f#"$dir"/}"
        case "$rest" in */*) ;; *) return 0 ;; esac ;;
    esac
  done <<< "$tracked"
  return 1
}

# 未被覆盖的仓库内包目录（每行一个，去重）
scan_missing() {
  local tracked p dir
  tracked="$(git ls-files -z | tr '\0' '\n')"
  while IFS= read -r p; do
    [ -n "$p" ] || continue
    dir="$(map_dir "$p")" || continue
    covered "$dir" "$tracked" || printf '%s\n' "$dir"
  done < <(collect_imports) | sort -u
}

# tracked（index）文件中 >1MB 的（输出 "大小<TAB>路径"）。
# 单次 git cat-file --batch-check 批量取 blob 大小，避免每文件起一个进程。
scan_big() {
  local pairs
  pairs="$(git ls-files -s -z | while IFS= read -r -d '' rec; do
    meta="${rec%%$'\t'*}"
    path="${rec#*$'\t'}"
    set -- $meta                       # mode sha stage
    printf '%s\t%s\n' "$2" "$path"     # sha<TAB>path
  done)"
  [ -n "$pairs" ] || return 0
  paste <(printf '%s\n' "$pairs" | cut -f1 \
            | git cat-file --batch-check='%(objectsize)') \
        <(printf '%s\n' "$pairs" | cut -f2-) \
    | awk -F'\t' -v lim="$LIMIT" '$1+0>lim {printf "%s\t%s\n", $1, $2}'
}

# ---- 自测：以 thunder/logs 为回归锚点 ----------------------------------
# 锚点含义：与本次真实缺陷同形的 import 路径
#   github.com/mszlu521/thunder/logs
# 在「源码未入库」时必须被判未覆盖；在「源码已入库」时必须被判已覆盖。
# 这样即便将来人把 thunder/logs/logs.go 真正 git add 了，锚点仍然有效。
self_test() {
  local fail=0
  local anchor='github.com/mszlu521/thunder/logs'
  local mapped absent_tracked present_tracked

  echo "[self-test] 回归锚点 = $anchor （→ thunder/logs，本次首个真阳性）"

  mapped="$(map_dir "$anchor")" || mapped=''
  if [ "$mapped" = 'thunder/logs' ]; then
    echo "  PASS 映射 $anchor -> $mapped"
  else
    echo "  FAIL 映射 $anchor -> '$mapped'（期望 thunder/logs）"; fail=1
  fi

  # 缺陷态：tracked 清单里没有 thunder/logs/logs.go
  absent_tracked="$(printf '%s\n' \
    'thunder/req/param.go' 'thunder/midd/cache.go' 'common/config/config.go' 'app/go.mod')"
  if covered 'thunder/logs' "$absent_tracked"; then
    echo "  FAIL 锚点缺陷态：未入库却判为已覆盖"; fail=1
  else
    echo "  PASS 锚点缺陷态：thunder/logs 被判为未覆盖（真阳性）"
  fi

  # 修复态：把源码加进 tracked 清单
  present_tracked="$(printf '%s\n%s\n' "$absent_tracked" 'thunder/logs/logs.go')"
  if covered 'thunder/logs' "$present_tracked"; then
    echo "  PASS 锚点修复态：thunder/logs 判为已覆盖"
  else
    echo "  FAIL 锚点修复态：已入库却判为未覆盖"; fail=1
  fi

  # 谓词边界：非仓库内包不参与（map_dir 返回 1）
  if map_dir 'github.com/gin-gonic/gin' >/dev/null 2>&1; then
    echo "  FAIL 外部依赖被误判为仓库内包"; fail=1
  else
    echo "  PASS 外部依赖不纳入检查"
  fi

  [ "$fail" -eq 0 ] && echo "[self-test] PASS" || echo "[self-test] FAIL"
  return "$fail"
}

if [ "${1:-}" = '--self-test' ]; then
  self_test
  exit $?
fi

# ---- 主检 ---------------------------------------------------------------
echo "[scope] 仓库内包 = app/common/core/model/thunder；>1MB 阈值 = ${LIMIT} bytes"

missing="$(scan_missing)"
if [ -n "$missing" ]; then
  while IFS= read -r d; do
    [ -n "$d" ] || continue
    cause="$(git check-ignore -v "$d/__probe__.go" 2>/dev/null || true)"
    if [ -n "$cause" ]; then
      printf '  MISSING  %s   (.gitignore 仍在忽略: %s)\n' "$d" "$cause"
    else
      printf '  MISSING  %s\n' "$d"
    fi
  done <<< "$missing"
fi
n_missing="$(printf '%s\n' "$missing" | grep -c . || true)"
printf '缺失 tracked 的本地 import 包 = %s （期望 0）\n' "$n_missing"

big="$(scan_big)"
if [ -n "$big" ]; then
  while IFS= read -r line; do
    [ -n "$line" ] || continue
    printf '  BIG  %s\n' "$line"
  done <<< "$big"
fi
n_big="$(printf '%s\n' "$big" | grep -c . || true)"
printf '>1MB tracked 文件 = %s （期望 0）\n' "$n_big"

if [ "$n_missing" -gt 0 ] || [ "$n_big" -gt 0 ]; then
  echo "FAIL: check-tracked-imports 未通过" >&2
  exit 1
fi
echo "OK: check-tracked-imports 通过"
