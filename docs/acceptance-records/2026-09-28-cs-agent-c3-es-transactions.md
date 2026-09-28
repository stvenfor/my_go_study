# Acceptance — cs-agent-c3-es-transactions

- Date: 2026-09-28
- Mode: **Partial**
- harness: 机跑 `go build` + 定向 `go test`；lan-up 创建→搜索人证 / ES down 写不 500 待人证

## 机跑

- `go build -o /tmp/my_go_study_api ./cmd/api` ✅
- `go test ./internal/repository/es/... ./internal/usecase/ -count=1 -run 'Transaction|Elastic|Search'` ✅

## 已交付

- compose：单节点 `elasticsearch`（security off）；lan-up 默认 `ELASTICSEARCH_URL=http://elasticsearch:9200`
- `internal/repository/es`：最小 HTTP REST（无官方 client）；mapping + Index/Delete/Search
- usecase：CUD 后 best-effort 同步索引；失败日志 + `es_transaction_index_failures_total`；不回滚写路径
- `GET /api/v1/transactions/search?q=` SessionAuth，强制会话 `user_id`
- 配置 `ELASTICSEARCH_URL`；文档 `docs/elasticsearch-transactions.md`
- `scripts/reindex_transactions_es.sh`（仅 mapping；全量灌数 Deferred）

## Deferred / 人证

- `make lan-up`：带 note 创建 → search 命中
- ES 停机后再 Create：仍 2xx
- 全量 PG→ES reindex
