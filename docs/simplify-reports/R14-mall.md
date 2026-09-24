# R14 — Mall（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R14 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. Before：`ListOnShelfProducts` / `MallProductWithSKUs` 无调用方（HTTP 用 ListShelfItems）。
## 2. Goals：G1 删除死路径（usecase/repo/iface/type）。
## 3. After：Go mall 层删除；`go build`/`go test usecase` 绿。

