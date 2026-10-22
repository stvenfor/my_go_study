# my_go_study 项目长期记忆

## 环境 / 工具链坑（本机，跨会话有效）
- **sandbox 的 `sed` 是 toybox**（WorkBuddy shim），`sed -i ''` 会坏；且 `BASH_ENV` 会重新注入 shim，普通改 PATH 无效。跑 `make` / 项目脚本时用：
  `env -u BASH_ENV PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/go/bin:$HOME/.local/bin" make ...`
- **Clash(127.0.0.1:7897) 会拒绝**这些国内镜像域（经 Clash `http=000`，直连 200）：`goproxy.cn`/`proxy.golang.org`/`goproxy.io`/`mirrors.aliyun.com`/`dl-cdn.alpinelinux.org`，以及 Docker Hub 的 `auth.docker.io`。Docker 构建/拉取需对它们**绕过代理**。
- **Docker registry-mirrors 写 `~/.docker/daemon.json` 能持久**（`docker.m.daocloud.io` 直连可达、自带鉴权，绕开 `auth.docker.io`）；而 `proxies` 写 daemon.json 会被 Docker Desktop 重启冲掉（只认 GUI）。

## make lan-up（Docker 全栈）跑通要件
- 镜像源：daemon.json `registry-mirrors=["https://docker.m.daocloud.io"]`；docker.io 镜像**逐个顺序拉**（daocloud 并发限流 `请求过于频繁`）。
- 构建：Dockerfile 已对 `goproxy.cn`+`dl-cdn.alpinelinux.org` 设 `NO_PROXY` 绕过 Clash；跑时传 `GOPROXY=https://goproxy.cn`。
- 工作区：`go.work`（go 1.25.0，use common/core/model/app/thunder）+ 空 `go.work.sum`，二者 gitignored 需本地生成。
- 冷启动：app/worker 已加 `restart: unless-stopped`，抵御 RabbitMQ AMQP 监听慢一拍导致的偶发 Exited(1)。

## 端口 / 访问
- HTTP :8080、gRPC :9090；PG :5432、Redis :6379、RabbitMQ :5672(管理 15672)、ES :9200；Prometheus :9091、Grafana :3000、Alertmanager :9093。局域网 IP 走 `REALTIME_PUBLIC_WS_HOST`（sync-lan-ip 自动写 .env.lan）。

## 约定
- 新业务表一律 `wys_` 前缀；改 Go 在本仓库、改 Flutter 进 `../../my_ai_project`；两仓独立 git。
- 交付物默认放 `~/Desktop/Matt/`（用户偏好，见用户级 MEMORY）。
- **多客户端同步局域网 IP**：`make lan-up LAN_CLIENTS=my_kmp_project,kmp_kuikly`（默认只写 `my_ai_project`；`LAN_CLIENTS=all` 写清单全部）。客户端清单在 `scripts/lan-ip.clients`，目录名即 ID，必须是 `../<name>` 同级目录；目录不存在只打印「跳过」不报错。
- ⚠️ `my_kmp_project` 的 `core/network/.../LanHost.kt` **已被 git 跟踪**，清单里的 `quote` 规则会把局域网 IP 写进这个源码文件、污染其工作区（正是清单头警告的反模式）。只想同步 `.env.lan` 就注释掉该 quote 行。`kmp_kuikly` 的 LanHost.kt 尚不存在，quote 行会被跳过。
