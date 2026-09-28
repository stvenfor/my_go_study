# 局域网联调可观测（`make lan-up` 默认拉起）

`make lan-up` / 基础 compose 会启动：

| 服务 | 容器内 | 本机访问 |
|------|--------|----------|
| API metrics | `app:8080/metrics` | `http://127.0.0.1:8080/metrics` |
| Prometheus | `:9090` | `http://127.0.0.1:9091` |
| Alertmanager | `:9093` | `http://127.0.0.1:9093` |
| Grafana | `:3000` | `http://127.0.0.1:3000`（默认 `admin` / `admin`） |
| feishu-adapter | `:8089`（仅内网） | 无宿主机端口 |

## 飞书告警

在 `.env.lan` 或环境中设置：

```bash
FEISHU_ALERT_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx
```

未设置时：Alertmanager 仍工作，adapter 对 webhook 返回 204 并丢弃（compose 不因缺 URL 失败）。

可选覆盖 Grafana 账密：`GRAFANA_ADMIN_USER` / `GRAFANA_ADMIN_PASSWORD`。

## 配置落点

- `deploy/observability/prometheus.yml` — scrape `app:8080/metrics`
- `deploy/observability/alerts.yml` — 5xx / target down / P99
- `deploy/observability/alertmanager.yml` — 路由到 feishu-adapter
- `deploy/observability/feishu_adapter.py` — AM → 飞书文本
- `deploy/observability/grafana/` — datasource + 基础 dashboard

## 冒烟

```bash
curl -s http://127.0.0.1:8080/metrics | head
# Prometheus UI → Status → Targets → job=app 应为 UP
```
