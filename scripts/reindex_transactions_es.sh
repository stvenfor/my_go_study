#!/usr/bin/env bash
# Partial：仅 PUT transactions 索引 mapping。全量从 Postgres 灌数 Deferred。
set -euo pipefail
URL="${ELASTICSEARCH_URL:-http://127.0.0.1:9200}"
URL="${URL%/}"
echo "Ensure index at $URL/transactions"
curl -fsS -X PUT "$URL/transactions" -H 'Content-Type: application/json' -d '{
  "mappings": {
    "properties": {
      "id": { "type": "long" },
      "user_id": { "type": "keyword" },
      "type": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "category": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "note": { "type": "text" },
      "amount": { "type": "double" },
      "date": { "type": "keyword" }
    }
  }
}' || true
echo
echo "OK (ignore resource_already_exists). Full PG→ES reindex is Deferred."
