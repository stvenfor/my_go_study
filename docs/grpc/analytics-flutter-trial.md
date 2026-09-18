# 数据分析 gRPC 联调（Flutter 试验）

> Flutter 首页「数据分析」→ 列表/详情走 Go BFF **gRPC :9090**；登录等仍走 HTTP `:8080`。

## 架构

```text
首页「数据分析」
  → AnalyticsListPage / AnalyticsDetailPage
  → gRPC AnalyticsService (:9090)
  → AnalyticsUsecase → Postgres analytics_records
```

## Go

| 项 | 值 |
|----|-----|
| 表 | `analytics_records`（≥30 字段，时间为 Unix 秒） |
| 种子 | 启动时 `EnsureSeedData`：若尚无 `seed_chart_v1` 标记则清空 `source_system=seed` 并写入约 48 条图表友好数据（漏斗递减、0–100 评分、异常/精选/零点击样例） |
| Proto | `api/proto/analytics/v1/analytics.proto` |
| 端口 | `grpc.port` 默认 `9090`（`GRPC_ENABLED` / `GRPC_PORT`） |
| 鉴权 metadata | `authorization`、`x-session-id`、`x-device-id` |

```bash
make proto   # 可选重生成
make run     # 本机进程：HTTP :8080 + gRPC :9090；自动刷新 chart-v1 种子
# 或 Docker 统一管理（推荐联调）：
make lan-up  # / make docker-up —— app 同时映射 8080 + 9090

# 强制重刷本地库种子（已有旧数据时）：
./scripts/load-env.sh go run ./cmd/seed-analytics

# 仅手工 SQL 最小集（可选）：
psql "$DATABASE_URL" -f scripts/seed_analytics_records_chart_v1.sql
```

冒烟（先 HTTP 登录拿 token）：

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -H "x-session-id: $SID" \
  -H "x-device-id: $DID" \
  -d '{"page":1,"page_size":10}' \
  127.0.0.1:9090 analytics.v1.AnalyticsService/ListAnalyticsRecords
```

## Flutter

| 路由 | 页面 |
|------|------|
| `/home/data_analytics` | 列表（分页） |
| `/home/data_analytics/detail` | 详情（全字段分区） |

- 入口：首页功能格「数据分析」（已有文案，已接 `AnalyticsNavigation`）
- Host：与 HTTP 同源（`BACKEND_HOST` / LAN）；端口 `BACKEND_GRPC_PORT`（默认 9090）

```bash
# 示例
flutter run --dart-define=BACKEND_HOST=172.16.0.43 --dart-define=BACKEND_GRPC_PORT=9090
```

## UI

Analytics Dashboard 色板：Primary `#1E40AF`，Accent `#D97706`，Background `#F8FAFC`；列表 `ListView.builder` + 空态/错误/加载。

## 相关

- [Gin HTTP vs gRPC](./gin-http-vs-grpc.md)
- 飞书：知识库 `tool` → 目录 `grpc`
