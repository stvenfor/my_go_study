#!/usr/bin/env bash
# 把 .githooks 安装到本仓库 .git/hooks（不改 git config）。
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
dest="$root/.git/hooks"
mkdir -p "$dest"
for hook in "$root"/.githooks/*; do
  name="$(basename "$hook")"
  cp "$hook" "$dest/$name"
  chmod +x "$hook" "$dest/$name"
done
echo "已安装钩子到 $dest"
