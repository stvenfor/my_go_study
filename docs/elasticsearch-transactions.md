# Elasticsearch 交易全文索引

开发/联调用单节点 ES 给 `transactions` 做 note/category/type 全文检索。不索引其它业务表。

## 启用

| 环境 | 配置 |
|------|------|
| `make lan-up` | compose 默认拉起 `elasticsearch`；app `ELASTICSEARCH_URL=http://elasticsearch:9200` |
| 本机 `make run` | 先起 ES（或只起 compose 里的 es），再设 `ELASTICSEARCH_URL=http://127.0.0.1:9200` |
| 关闭 | 不设 / 空 `ELASTICSEARCH_URL`（搜索 API 返回 503；CRUD 不受影响） |

环境变量：`ELASTICSEARCH_URL` → `config.elasticsearch.url`。

## 索引

- 名：`transactions`
- 字段：`id`、`user_id`（keyword）、`type`、`category`、`note`、`amount`、`date`
- 启动时 `EnsureIndex`；已存在则跳过

## 写路径

Create / Update / Delete 成功后**同步**写 ES。

- 失败：打 warn 日志 + `es_transaction_index_failures_total`；**HTTP 仍 2xx**（PG 已提交）
- `ponytail:` 无 outbox；升级路径 = 事务 outbox / 异步补偿

## 搜索 API

```http
GET /api/v1/transactions/search?q=午饭&limit=20
Authorization: Bearer <access>
X-Session-ID: ...
X-Device-ID: ...
```

- SessionAuth；过滤**仅**会话 `user_id`（不可跨用户）
- 全文：`note`、`category`、`type`
- 响应：`{ "items": [ ... ] }`（与 Flutter list 风格一致）

## 简易 reindex（Partial）

```bash
./scripts/reindex_transactions_es.sh
# 或 ELASTICSEARCH_URL=http://127.0.0.1:9200 ./scripts/reindex_transactions_es.sh
```

脚本只确保 mapping；**全量从 PG 灌索引 Deferred**（可随后用登录态创建/更新触发增量，或另写导入）。

## 人证（lan-up）

1. 登录后 `POST /api/v1/transactions` 带 note
2. `GET /api/v1/transactions/search?q=<note片段>` 命中
3. 停掉 ES 容器后再 Create：应仍 201，日志有 index failed
