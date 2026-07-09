# Transactions（二手车/收支）初学者逐行导读

> 登录后的 Flutter 通过 Go BFF 访问 Supabase `transactions` 表，**双层隔离**：Go 强制 `user_id` 过滤 + Supabase RLS。

---

## 0. 数据流一图

```
Flutter TransactionApi
  → GET /api/v1/transactions?limit=20&offset=0
  → Header: Authorization: Bearer <Supabase token>
  → Go SupabaseAuth 中间件校验 token，取出 user.ID
  → TransactionController → TransactionUsecase → Supabase PostgREST
  → .Eq("user_id", userID)  // 应用层过滤
  → Supabase RLS            // 数据库层过滤（需执行迁移 SQL）
  → JSON { "items": [...] } 或 ResultModel 信封
```

---

## 1. 推荐阅读顺序

### Go

| 顺序 | 文件 | 内容 |
|------|------|------|
| 1 | `internal/domain/entity/transaction.go` | 表字段模型 |
| 2 | `internal/usecase/transaction_usecase.go` | 薄用例层 |
| 3 | `internal/repository/supabase/transaction_repo.go` | PostgREST 查询 |
| 4 | `internal/delivery/http/controller/transaction_controller.go` | 两套 API 风格 |

### Flutter

| 顺序 | 文件 | 内容 |
|------|------|------|
| 1 | `packages/features/home/lib/home/model/transaction_model.dart` | fromJson |
| 2 | `packages/features/home/lib/home/api/transaction_api.dart` | HTTP + 分页 |
| 3 | `packages/features/home/lib/home/repository/transaction_repository.dart` | 解包 PageResult |

---

## 2. 为什么有两套列表 API？

| 路径 | 分页参数 | 响应格式 | 谁在用 |
|------|----------|----------|--------|
| `GET /api/v1/transactions` | `limit` + `offset` | `{ "items": [...] }` 直出 | **Flutter 二手车列表** |
| `GET /api/v1/transactions/manage` | `page` + `size` | ResultModel + pagination | 管理端/统一风格 |

Controller 的 `List` 方法：若 URL 带 `page` 参数则走 manage，否则走 legacy。

---

## 3. 安全：user_id 为什么传两次？

Repository 每个方法签名都有 `(accessToken, userID, ...)`：

```go
query := client.From("transactions").
    Select("*", "", false).
    Eq("user_id", userID)  // 应用层：即使 RLS 未配也不串数据
```

**accessToken** 用于 `WithUserToken`：PostgREST 以**用户身份**请求，RLS 策略生效。

**userID** 用于显式过滤：防御性编程，代码审查一眼可见隔离逻辑。

---

## 4. Flutter 分页约定

```dart
// Controller 用 0-based page
// Api 层：offset = page * size
queryParameters: {
  'limit': size,
  'offset': page * size,
}
```

Repository 返回 `PageResult`，UI 不需要知道 `ResultModel` 结构。

---

## 5. 列表响应示例

**Flutter 兼容接口**

```json
{
  "items": [
    {
      "id": 1,
      "user_id": "uuid",
      "type": "expense",
      "category": "food",
      "amount": 88.5,
      "date": "2026-07-09"
    }
  ]
}
```

**401 未授权**：token 过期 → Flutter 提示重新登录。

---

## 6. RLS 迁移（必做）

未执行 SQL 迁移时，直连 PostgREST 仍可能越权。见：

- `supabase/migrations/003_transactions_user_id_rls.sql`
- `./scripts/check_transactions_rls.sh`

---

## 7. 本地调试

```bash
# 先登录拿 TOKEN
TOKEN=$(curl -sf -X POST http://127.0.0.1:8080/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"you@example.com","password":"xxx"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])")

curl -s "http://127.0.0.1:8080/api/v1/transactions?limit=5&offset=0" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

或 `make test-transactions`。

---

## 8. 相关文档

- [认证初学者导读](./auth-beginner-walkthrough.md)
- [Supabase 集成说明](./supabase-integration.md)
- Flutter [BACKEND_INTEGRATION.md](../../../my_ai_project/docs/BACKEND_INTEGRATION.md) §5
