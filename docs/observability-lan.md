# 局域网联调可观测（`make lan-up` 默认拉起）

`make lan-up` / 基础 compose 会启动：

| 服务 | 容器内 | 本机访问 |
|------|--------|----------|
| API metrics | `app:8080/metrics` | `http://127.0.0.1:8080/metrics`（**本机进程**） |
| Prometheus | `:9090` | `http://127.0.0.1:9091` |
| Alertmanager | `:9093` | `http://127.0.0.1:9093` |
| Grafana | `:3000` | `http://127.0.0.1:3000`（默认 `admin` / `admin`） |
| feishu-adapter | `:8089`（仅内网） | 无宿主机端口 |

## `/metrics` 访问收口（h7）

`/metrics` 曾随 `8080:8080` 一起暴露到局域网。现在由
`app/internal/delivery/http/router/router.go` 的守卫收口为 **默认只本机可达**：

- 只认 TCP 对端 `RemoteAddr`，**不信** `X-Forwarded-For` / `ClientIP()`（防伪造）；
- loopback（`127.0.0.1` / `::1`）放行 —— `curl http://127.0.0.1:8080/metrics`、
  Prometheus 抓本机口均可用；
- 局域网 IP 一律 **403**（macOS Docker Desktop 下经 docker-proxy 端口映射进来的
  `192.168.65.1` 等网关也不放行；Linux userland-proxy 的差异见下）；
- Docker 内网抓取（`prometheus.yml` 的 `app:8080/metrics`）由 compose 注入的
  `METRICS_ALLOW_CIDRS` 显式放行，默认 `172.16.0.0/12,10.0.0.0/8`（覆盖 compose
  网桥网段，如本机 `docker_default` = `172.18.0.0/16`）。自定义 Docker 网段需自行覆盖该变量。

因此：

- `deploy/observability/alerts.yml` 的 4 条 payment 规则（依赖 `up{job="app"}`、
  `payment_*` 指标）仍由 Prometheus 从 Docker 内网正常抓取，收口未掐断监控；
- 局域网设备取不到指标；
- **Docker 部署下从宿主机 `curl 127.0.0.1:8080/metrics` 是否 403 取决于平台与网络栈，
  不是「端口映射必 403」**：端口映射的对端不是 loopback；
  - **Linux + Docker userland-proxy**：容器内 `RemoteAddr` 常是网桥网关（如 `172.17.0.1`），
    会命中默认 `METRICS_ALLOW_CIDRS` 的 `172.16.0.0/12` 而被**放行**（不是 403）。
    若不想放行，收紧 `METRICS_ALLOW_CIDRS` 到只覆盖 Prometheus 所在网段，或一律走容器内抓取；
  - **macOS Docker Desktop 实测**：对端为 `192.168.65.1`，不在放行网段，故 **403**。
  稳妥取法一律用 `docker exec my_go_study_app curl -s http://127.0.0.1:8080/metrics`
  或经 `127.0.0.1` 的容器内抓取；本机 `make run` 进程不受影响。

## 飞书告警

在 `.env.lan` 或环境中设置：

```bash
FEISHU_ALERT_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx
```

未设置时：Alertmanager 仍工作，adapter 对 webhook 返回 204 并丢弃（compose 不因缺 URL 失败）。

可选覆盖 Grafana 账密：`GRAFANA_ADMIN_USER` / `GRAFANA_ADMIN_PASSWORD`。

## 配置落点

- `deploy/observability/prometheus.yml` — scrape `app:8080/metrics`
- `deploy/observability/alerts.yml` — 5xx / target down / P99 + 4 条 payment
- `deploy/observability/alertmanager.yml` — 路由到 feishu-adapter
- `deploy/observability/feishu_adapter.py` — AM → 飞书文本
- `deploy/observability/grafana/` — datasource + 基础 dashboard
- `docker/docker-compose.yml`（app 服务 `METRICS_ALLOW_CIDRS`）— 仅放行 Docker 内网抓取

## 冒烟

```bash
# 本机进程（make run / go run）：loopback 可取到指标
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/metrics   # 200
# 局域网 IP：收口后取不到
curl -s -o /dev/null -w '%{http_code}\n' http://<本机LAN_IP>:8080/metrics # 403 / 不可达

# Docker 部署：容器内 loopback
docker exec my_go_study_app curl -s http://127.0.0.1:8080/metrics | head

# Prometheus UI → Status → Targets → job=app 应为 UP
```
