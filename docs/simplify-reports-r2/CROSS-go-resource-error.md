# CROSS — 社区/短视频错误映射

| 字段 | 值 |
|------|-----|
| ID | CROSS |
| Program | Round-2 deep simplify |
| 状态 | Done |
| 硬约束 | 业务/页面语义不变 |

## 1. Before
community / short_video controller 同构 writeXxxError。
## 2. Goals
G1 writeResourceCRUDError 共享（文案不变）。
## 3. After
`resource_error.go`(+test)；两 controller 委托。

