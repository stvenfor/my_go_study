# R19 — JPush / devices（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R19 |
| 状态 | Done（第二轮） |
| Ponytail | 已过滤 |

## 1. Before
- `SendInput.PreferRegistration` 无任何调用方置 true（HTTP/NotifyUser 均未传）→ `ListByAlias` 分支死代码。
- `PushDeviceRepository.ListByUser` / `ListByAlias` 仅服务上述死分支。

## 2. Goals
- G1：删 PreferRegistration 字段与死分支。
- G2：iface/repo/test mock 去掉 ListByUser、ListByAlias。

## 3. After
| Goal | 改动 | 不变 | 验证 |
|------|------|------|------|
| G1–G2 | `jpush_usecase.go`、`push_device_repository.go`、`push_device_repo.go`、test | 是（现行发送路径仍按 alias/rids） | `go test -run JPush` + `go build ./cmd/api` |

### 3.3 Skipped：sms/jpush Worker stub、Flutter linking 层（有意）
