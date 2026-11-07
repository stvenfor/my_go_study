# Slice — harden-h8-membership-settler

## Slice Brief
- SOURCE_MODULE: `app`（`internal/payment`、`internal/membership`）
- TARGET_MODULE: 同上
- Source entry: P0-2 残留 **F2** —— `entity.PaymentRefMembershipOrder`（`model/entity/payment.go:34`）在 `payment.Service` 的 settlers 表里**没有注册者**，核销成功只会走 `settled_no_settler` 分支（`app/internal/payment/settle.go:103-116`）打 Warn；渠道买会员**必须**靠客户端回跳 `ConfirmBuyout` 才发权益，回跳丢失/用户杀进程 → 付了钱没会员
- Target entry: 回调（或查单补）核销即主动发放权益；与 `ConfirmBuyout` 双链并发只发一次
- 本轮 ONLY: ①给 `membership.Service` 实现 `payment.Settler`（`SettleChannelPaid`）②在 `membership.NewService` 内自注册（照抄 `mall/service.go:44-52` 的 `WithPayment` 形态）③幂等 + 并发证明
- **零 `cmd/api/main.go` 改动**（取证更正：`main.go:246` 已把 `paymentSvc` 传进 `membership.NewService`，注册在包内完成即可；Program 冲突矩阵里「h8 独占 main.go」的假设作废）
- 不做: 不改 notify 验签逻辑；不动 `payment` 的置 paid 事务边界；不顺手做 F7（核销后通知用户，另列）；不改 Flutter
- 验收:
  - [ ] `payment.Settler` 由 `membership.Service` 实现，`ref_type=membership_order` 核销后 entitlement 生效；`settled_no_settler` 对 membership_order 不再出现（负例：断言该 reason 计数/日志为 0）
  - [ ] 幂等：同一 `refID` 连投两次回调 → 权益只延长一个周期（复用 `ExtendEntitlement` 现有语义；若不足以幂等则本 Slice 必须给出证据并说明怎么保证）
  - [ ] **双链并发真库用例**（风险 R5）：一条走 `SettleChannelPaid`、一条走 `ConfirmBuyout`，同一 order 并发跑 → 断言只有一条 entitlement 周期被加上、`wys_membership_entitlement` 不出现重复叠加。放 `app/internal/membership/`，`//go:build integration`，scratch 库 `P03_PG_DBNAME`（默认 `p03_it`），**绝不连业务库 `my_go_study`**
  - [ ] `ConfirmBuyout` 的「必须有已核销流水」（`payment.SettledRecord`，`settle.go:36-43`）在 h8 之后仍成立，且不发第二次权益 —— 对照测试
  - [ ] 事务边界不破：`SettleChannelPaid` 必须在 payment 传入的 ctx 事务里写（`.harness/rules` + `settle.go:1-6` 的「核销与发货同事务」约定），体内零外部 HTTP/ES/MQ
  - [ ] `go -C app test ./internal/{payment,membership}/ -count=1` 绿 + integration 用例绿 + `go build all` 绿
  - [ ] 更新 `docs/placeholder-inventory.md` 第 18 条（`membership_order` 没有业务核销器）与 `docs/acceptance-records/2026-11-04-parity-p0-2-payment-notify-ledger.md` §七 F2 状态
- 文件白名单:
  - `app/internal/membership/service.go`
  - `app/internal/membership/{settle.go,repository.go,repo_postgres.go}`（按需）
  - `app/internal/membership/*_test.go`
  - `docs/placeholder-inventory.md`（追加/就地更新对应条目）
  - `docs/acceptance-records/2026-11-05-harden-h8-membership-settler.md`
  - `plans/slices/harden-h8-membership-settler.md`（仅 Context Card）
- 文件黑名单:
  - `app/cmd/api/main.go`（本 Slice 不需要；若发现「确实需要」，停下问人）
  - `app/internal/payment/{notify.go,signer*.go}`（验签逻辑冻结）
  - `app/internal/mall/**`
  - `**/go.mod`、`**/go.sum`
- 验证命令:
  - `go -C app test ./internal/payment/ ./internal/membership/ -count=1`
  - `GOPROXY=https://goproxy.cn,direct go -C app test -tags=integration ./internal/membership/ -run 'Settle|Buyout' -count=1`
  - `go build all`
  - 人工冒烟（8081 + worker）：造一条 `membership_order` pending 流水 → 模拟回调 → 查 `wys_membership_entitlement` 是否出现，且重复回调不叠加
- 证据: `docs/acceptance-records/2026-11-05-harden-h8-membership-settler.md`

## Context Card — harden-h8-membership-settler
- 已完成: `membership.Service` 实现 `payment.Settler`（`SettleChannelPaid`）；`NewService` 内自注册 `PaymentRefMembershipOrder`（零 main.go 改动）；`var _ payment.Settler` 编译期断言；幂等 + 双链并发真库用例（`p08_it`）；单测 3 条 + integration 4 条全绿；`go build all`(go1.26.4)/vet 通过；文档（清单条目 18 / parity §七 F2 / 本片验收记录）已更新。证据：`docs/acceptance-records/2026-11-05-harden-h8-membership-settler.md`
- 未做/Deferred: 8081+worker 人工端到端冒烟（Partial Deferred，沿上一轮人证口径）；F7 核销后通知；Flutter。
- 关键文件: `app/internal/membership/service.go`（`NewService` 自注册 / `SettleChannelPaid`）；`app/internal/membership/service_test.go`；`app/internal/membership/settle_integration_test.go`（新）；`app/internal/payment/settle.go:103-116`（兜底分支保留）。
- harness: post ok? 是 —— 单测 + integration（含 `-count=10` 并发）+ `go build all`(1.26.4) + vet 全绿；HEAD `e288553` / worktree `harden-h8` 自证通过；未 commit/push、未改 go.mod/go.sum、业务库只读。
- 下一 Slice 建议: payment 侧刷新 `settled_no_settler` 分支陈旧注释（仍举 membership_order 为例、称其仓储不支持 txdb）；F7 事务提交后入队通知；8081+worker 端到端冒烟。
- 已知坑: 默认 go1.27.3 `go build all` 红在 grpc v1.71.0 `handler_server.go:271 undefined http2.TrailerPrefix`（共同前提，非本片）；改用 `/usr/local/go/bin/go`(1.26.4)。`SettleChannelPaid` 内**不要**调 `u.repo.Transaction`（那会在 base db 另开事务、脱离 payment ctx），直接 `MarkOrderPaid`/`ExtendEntitlement` 经 `txdb.DB` 落同一事务。integration 用独立 scratch `p08_it` 避开 `p03_it` 建表冲突。
