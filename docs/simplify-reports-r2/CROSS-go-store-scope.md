# CROSS — Go 门店作用域/分页共享

| 字段 | 值 |
|------|-----|
| ID | CROSS |
| Program | Round-2 deep simplify |
| 状态 | Done |
| 硬约束 | 业务/页面语义不变 |

## 1. Before
deal_invoice / used_car / new_car_follow / after_sales 重复 requireCurrentStore、pageOffset、status 白名单、image URL。
## 2. Goals
G1 抽 store_scope.go + 表驱动测试；调用方语义不变。
## 3. After
新建 `internal/usecase/store_scope.go`(+test)；四 usecase 改用共享助手。~−150 行重复。
验证：go test ./internal/usecase

