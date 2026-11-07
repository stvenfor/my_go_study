# Acceptance Record — harden-h8-membership-settler（P0-2 残留 F2）

- 日期：2026-11-05（工作树 HEAD `e288553`，worktree `harden-h8`）
- 来源残留：`docs/placeholder-inventory.md` 条目 18；
  [`2026-11-04-parity-p0-2-payment-notify-ledger.md`](./2026-11-04-parity-p0-2-payment-notify-ledger.md) §七 F2；
  [`2026-11-04-membership-money-tx-followup.md`](./2026-11-04-membership-money-tx-followup.md) §诚实口径 3
- 主交付物：给 `membership.Service` 实现 `payment.Settler`（`SettleChannelPaid`），
  在 `membership.NewService` 内自注册为 `membership_order` 核销器，**零 `cmd/api/main.go` 改动**
- 关联：`plans/slices/harden-h8-membership-settler.md`；Program `backend-hardening` Task `task_92adb13b2632`

## status: **Migrated**（F2 已收口）

## 改了什么

| 文件 | 改动 |
|------|------|
| `app/internal/membership/service.go` | ① 加 `var _ payment.Settler = (*Service)(nil)` 编译期断言；② `NewService` 内 `pay != nil` 时 `pay.RegisterSettler(entity.PaymentRefMembershipOrder, u)`（照 mall `WithPayment` 形态，pay 本就由 main 注入构造器，无需动 main.go）；③ 新增 `SettleChannelPaid`：`refType`/`refID`/归属校验 → `MarkOrderPaid` → `ExtendEntitlement`。**不另开事务**，直接复用 payment 传入的 ctx 事务句柄 |
| `app/internal/membership/service_test.go` | 内存夹具补 3 条单测：正常发权益、连投两次幂等（只一个周期）、refType 非法 / 流水归属不符拒绝 |
| `app/internal/membership/settle_integration_test.go`（新，`//go:build integration`） | 真库 4 条：回调核销即发权益 + `Reason=settled`（负例：非 `settled_no_settler`）+ 重复回调幂等 + 回调后 `ConfirmBuyout` 不发第二周期；`SettleChannelPaid` vs `ConfirmBuyout` 真库双链并发；对照「无已核销流水仍拒绝」；事务边界（强制外层回滚证明写在同一 ctx 事务） |
| `docs/placeholder-inventory.md` | 条目 18 `Degraded → Migrated`，速览表 + 正文同步 |
| `docs/acceptance-records/2026-11-04-parity-p0-2-payment-notify-ledger.md` | §七 F2 改为「已解除（Migrated）」 |

## 设计要点 / 事务边界

- **自注册为何在包内**：`main.go:246` 已把 `paymentSvc` 传进 `membership.NewService`，
  与 mall 的 `WithPayment` 等价形态在构造器里一行 `RegisterSettler` 即可，
  故 Program 冲突矩阵里「h8 独占 main.go」的取证更正成立 —— 本片 `git diff` 不含 `main.go`。
- **幂等 / 并发守卫**：`SettleChannelPaid`（回调链）与 `ConfirmBuyout`（回跳链）都以
  `MarkOrderPaid` 的 `WHERE status=pending + RowsAffected==0 → ErrOrderPaid` 为原子守卫；
  只有抢到行锁的那条走到 `ExtendEntitlement`，另一条 `ErrOrderPaid` 直接幂等返回 / 递归读权益。
  payment 侧 `SettlePaid` 的 `newly` 标志又给「同一回调重投」加了第二层幂等。
- **同事务**：`payment.settleByFact` 在 `repo.WithTx` 内先 `SettlePaid` 再调 `SettleChannelPaid`，
  传入的 ctx 已带 `txdb` 事务句柄；本实现**不调用 `u.repo.Transaction`**（那会在 base db 上另开事务、脱离 payment 事务），
  `MarkOrderPaid` / `ExtendEntitlement` 经 `txdb.DB(ctx, ..)` 直接落进该事务；
  `ExtendEntitlement` 内层 `Transaction` 落为 SAVEPOINT。任一步失败连流水一起回滚为 pending。
- **体内零外部调用**：`SettleChannelPaid` 只读写 membership 两张表，无 HTTP / ES / MQ / 推送
  （沿用 P0-3 立的规矩；F7「核销后通知」须在事务提交后入队，本片不做）。
- `GetOrder` 是事务外的普通 SELECT（读已提交的业务单），不参与守卫；守卫是 `MarkOrderPaid` 的条件更新。

## 验证

```bash
go version           # go1.26.4 darwin/amd64（用 /usr/local/go/bin/go）
# 默认 go1.27.3 下 `go build all` 仍红：grpc v1.71.0 handler_server.go:271 undefined http2.TrailerPrefix
# —— 属共同前提里的既知外部问题，非本片引入（实测确认，见下）。

go -C app test ./internal/payment/ ./internal/membership/ -count=1
go -C app vet -tags=integration ./internal/membership/

# 先建 scratch 库（绝不连业务库 my_go_study）
docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p08_it"
DATABASE_PASSWORD=postgres P08_PG_DBNAME=p08_it \
  go -C app test -tags=integration ./internal/membership/ -run 'Settle|Buyout' -count=1 -v
DATABASE_PASSWORD=postgres P03_PG_DBNAME=p03_it P08_PG_DBNAME=p08_it \
  go -C app test -tags=integration ./internal/membership/ -count=1   # 含余额真库用例，零建表冲突
DATABASE_PASSWORD=postgres P08_PG_DBNAME=p08_it \
  go -C app test -tags=integration ./internal/membership/ -run TestSettleChannelPaidVsConfirmBuyoutConcurrent -count=10

go build all         # go1.26.4 退出 0
```

实测输出（go1.26.4）：

```
ok  app/internal/payment       # unit
ok  app/internal/membership    # unit
ok  app/internal/membership    # -tags=integration 全量（p03_it + p08_it）
ok  app/internal/membership    # 双链并发 -count=10（无 flaky）
VET_OK
BUILD_ALL_OK

--- PASS: TestSettleChannelPaidGrantsEntitlementAndIdempotent   # Reason=settled、重复回调 duplicate、权益 1 行且只 +1 周期、回调后 ConfirmBuyout 不叠加
--- PASS: TestSettleChannelPaidVsConfirmBuyoutConcurrent        # 双链并发只加一个周期，wys_membership_entitlement 不重复叠加
--- PASS: TestConfirmBuyoutRequiresSettledRecord                # 无流水仍 ErrPaymentNotConfirmed、不发权益、订单仍 pending
--- PASS: TestSettleChannelPaidSharesPaymentTx                  # 强制外层回滚 → 权益/订单随之消失（证明写在同一 ctx 事务）
--- PASS: TestSettleChannelPaidGrantsEntitlement / ...Idempotent / ...Rejects  # 内存夹具单测
```

`git status --short`（工作树仅 3 个文件，全部在白名单内，未碰任何 `**/go.mod`、`main.go`、
`app/internal/mall/**`、`payment/{notify.go,signer*.go}`；未 `git add` 任何未跟踪构建前置）：

```
 M app/internal/membership/service.go
 M app/internal/membership/service_test.go
?? app/internal/membership/settle_integration_test.go
```

（docs 改动另计：placeholder-inventory.md、parity-p0-2 验收记录、本文件、Slice Context Card。）

## 诚实口径 / 已知残留

1. **负例不是空断言**：`TestSettleChannelPaidGrantsEntitlementAndIdempotent` 断言
   `res.Reason == "settled" && res.NewlySettled == true`；若自注册没生效、
   仍走 `settled_no_settler` 分支，则 `Reason`/`NewlySettled` 必然不同 → 用例必挂。
   该断言直接覆盖「`settled_no_settler` 对 membership_order 不再出现」。
2. **并发用例是两条不同的链**：链 A 用 `payRepo.WithTx` 复刻 `settleByFact` 的事务 ctx 后直接调
   `SettleChannelPaid`，链 B 调 `ConfirmBuyout`（其 `ConfirmRefPaid` 因预置已核销流水直接返回，不触发渠道 HTTP）。
   两条链各自开事务、在同一 order 行上争夺 `MarkOrderPaid` 条件更新；`-count=10` 无 flaky。
3. **事务边界用例是「写后强制回滚」**：`SettleChannelPaid` 成功后让外层 `WithTx` 返回错误，
   断言权益 / 订单随之消失。若错误实现在 `SettleChannelPaid` 里另开独立事务，写会残留 → 用例必挂。
   这比单纯「成功」更能证明「写在同一 payment ctx 事务」。
4. **窗口断言口径**：`h8AssertOnePeriod` 用 `expires ∈ (now+20d, now+45d)` 区分「+1 周期(≈30d)」
   与「+2 周期(≈60d)」，与具体月长 / 时刻无关；叠加两次必落窗外。
5. **未做 / 不在白名单（显式登记，不静默丢弃）**：
   - `payment/settle.go:105-108` 的注释仍把 `membership_order` 当「未登记」示例、且称
     「membership 仓储还不支持 txdb」—— 事实已变（注释陈旧），但分支逻辑作为「未知 ref_type 兜底」仍正确。
     payment 包不在 h8 白名单，且 Slice 要求不动 payment 的置 paid 事务边界，故留给后续清理。
   - F7「核销后通知用户」仍 `Deferred`，与本片无交集。
   - 未跑 8081 + worker 的人工端到端冒烟（Slice 验证命令第 4 条）与 Flutter 改动（不在范围）：
     `curl` 级冒烟仍为 Partial Deferred，理由同沿上一轮（本机未起 worker / 未造渠道回调）。
6. **零 commit / push / rm -rf**，业务库 `my_go_study` 全程只读，DB 仅用 scratch `p03_it` / `p08_it`。

## 下一步

- payment 侧刷新 `settled_no_settler` 分支注释（去掉陈旧的 membership_order / txdb 描述）。
- F7：核销事务提交后入队通知（事务内零外部调用）。
- 8081 + worker 端到端冒烟（造 pending `membership_order` 流水 → 模拟回调 → 查 `wys_membership_entitlement`）。
