#!/usr/bin/env bash
# 按当前默认路由网卡的 IPv4，同步 Go 与清单里的全部客户端。
# 客户端清单：scripts/lan-ip.clients（Flutter / KMP / RN / Kuikly / uni-app）。
# make lan-up / make lan-run、Flutter ./scripts/run_app.sh --lan、IDE「LAN 真机」启动前会自动调用。
# 若 Docker app 已在跑且 IP 过期，本脚本会重建该容器（不重新编译镜像）。
# lan-compose 自己会 up，调用时设 SYNC_LAN_IP_SKIP_DOCKER=1 避免重复重建。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

detect_lan_ip() {
  local iface="" ip="" cand
  iface="$(route -n get default 2>/dev/null | awk '/interface:/{print $2; exit}')"
  if [[ -n "$iface" ]]; then
    ip="$(ipconfig getifaddr "$iface" 2>/dev/null || true)"
  fi
  if [[ -z "$ip" ]]; then
    for cand in en0 en1 en2; do
      ip="$(ipconfig getifaddr "$cand" 2>/dev/null || true)"
      if [[ -n "$ip" ]]; then
        break
      fi
    done
  fi
  if [[ -z "$ip" || "$ip" == 169.254.* ]]; then
    echo "错误: 未能探测到局域网 IPv4。请确认 Mac 已连接 Wi-Fi。" >&2
    return 1
  fi
  printf '%s\n' "$ip"
}

upsert_kv() {
  local file="$1" key="$2" value="$3"
  if [[ ! -f "$file" ]]; then
    echo "跳过（文件不存在）: $file" >&2
    return 0
  fi
  if grep -q "^${key}=" "$file"; then
    sed -i '' "s|^${key}=.*|${key}=${value}|" "$file"
  else
    printf '\n%s=%s\n' "$key" "$value" >>"$file"
  fi
}

IP="$(detect_lan_ip)"
echo "探测到局域网 IP: $IP"

GO_ENV="$ROOT/.env.lan"
if [[ ! -f "$GO_ENV" ]]; then
  if [[ -f "$ROOT/.env.lan.example" ]]; then
    cp "$ROOT/.env.lan.example" "$GO_ENV"
    echo "已从 .env.lan.example 创建 $GO_ENV"
  else
    echo "错误: 缺少 $GO_ENV" >&2
    exit 1
  fi
fi
OLD_GO="$(grep -E '^REALTIME_PUBLIC_WS_HOST=' "$GO_ENV" | head -1 | cut -d= -f2- || true)"
upsert_kv "$GO_ENV" REALTIME_PUBLIC_WS_HOST "$IP"
echo "Go REALTIME_PUBLIC_WS_HOST: ${OLD_GO:-<空>} → $IP"

python3 - "$ROOT" "$IP" "$ROOT/scripts/lan-ip.clients" <<'PY'
import pathlib, re, sys
root, ip, manifest = sys.argv[1], sys.argv[2], sys.argv[3]
root_path = pathlib.Path(root)
for raw in pathlib.Path(manifest).read_text().splitlines():
    line = raw.strip()
    if not line or line.startswith("#"):
        continue
    kind, rel_root, rel_file, marker = line.split("\t")
    client = (root_path / rel_root).resolve()
    if not client.is_dir():
        print(f"跳过（目录不存在）: {rel_root}")
        continue
    path = client / rel_file
    label = f"{client.name}/{rel_file}"
    if kind == "env":
        if not path.exists():
            example = client / ".env.lan.example"
            if example.exists():
                path.write_text(example.read_text())
            else:
                path.write_text(f"# 由 sync-lan-ip 维护，勿手填固定 IP\n{marker}=\n")
        text = path.read_text()
        old = ""
        matched = re.search(rf"^{re.escape(marker)}=(.*)$", text, re.M)
        if matched:
            old = matched.group(1)
            text = re.sub(rf"^{re.escape(marker)}=.*$", f"{marker}={ip}", text, count=1, flags=re.M)
        else:
            if text and not text.endswith("\n"):
                text += "\n"
            text += f"{marker}={ip}\n"
        path.write_text(text)
        print(f"{label} {marker}: {old or '<空>'} → {ip}")
        continue
    if kind != "quote":
        raise SystemExit(f"未知 kind: {kind}")
    if not path.exists():
        print(f"跳过（文件不存在）: {label}")
        continue
    quote = marker[-1]
    if quote not in ("'", '"'):
        raise SystemExit(f"quote 规则必须以引号结尾: {label}")
    text = path.read_text()
    pattern = re.escape(marker) + r"[^" + quote + r"]*" + re.escape(quote)
    new, n = re.subn(pattern, marker + ip + quote, text, count=1)
    if n != 1:
        raise SystemExit(f"未能更新 {label}")
    path.write_text(new)
    print(f"{label} → {ip}")
PY

refresh_running_app() {
  if [[ "${SYNC_LAN_IP_SKIP_DOCKER:-}" == "1" ]]; then
    return 0
  fi
  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi
  if ! docker inspect my_go_study_app >/dev/null 2>&1; then
    return 0
  fi
  local running current
  running="$(docker inspect -f '{{.State.Running}}' my_go_study_app 2>/dev/null || true)"
  if [[ "$running" != "true" ]]; then
    return 0
  fi
  current="$(docker exec my_go_study_app printenv REALTIME_PUBLIC_WS_HOST 2>/dev/null || true)"
  if [[ "$current" == "$IP" ]]; then
    echo "Docker app 已是 ${IP}，无需重建"
    return 0
  fi
  echo "Docker app 仍是 ${current:-<空>}，按 $IP 重建容器（不重新编译镜像）..."
  # shellcheck disable=SC1091
  source "$ROOT/scripts/source-env.sh"
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env.lan"
  set +a
  export APP_ENV="${APP_ENV:-lan}"
  docker compose \
    -f "$ROOT/docker/docker-compose.yml" \
    -f "$ROOT/docker/docker-compose.lan.yml" \
    up -d --no-deps --no-build app
  echo "Docker app REALTIME_PUBLIC_WS_HOST → $IP"
}

refresh_running_app
