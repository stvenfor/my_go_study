# Acceptance Record — parity-p0-3-money-tx-atomic

- 日期：2026-11-04
- Slice：`plans/slices/parity-p0-3-money-tx-atomic.md`
- Epic：`plans/epics/xhs-parity-p0-correctness.md` §P0-3
- 主交付物：`app/internal/txdb/`（context 传事务句柄）+ mall 支付/取消两条资金链进同一事务
  + 真库（PG16）回滚/并发用例（`-tags=integration`）

## status: **Migrated**（mall 域资金原子性）；曾登记一条同类残留在 membership 域，**同日已解除**（见末尾与 follow-up 记录）

note：

1. 原子性由**数据库事务**保证，补偿式回滚（`restorePayDebits`）整块删除 —— 不是「把补偿写得更好」，
   是「不再需要补偿」。
2. 顺手修掉一个此前没人点名的**双重扣款 bug**：旧实现对已支付订单重复调 `PayOrder` 会**再扣一次钱包**，
   然后 `PayOrderLocal` 幂等返回「已支付」，钱却少了。新实现先锁订单行、已结算就跳过扣减
   （并发用例 `TestMoneyTxConcurrentPayDebtsOnce` 锁死这条行为）。
3. 顺带修掉退款静默吞钱：旧 `CancelOrder` 里金额解析失败或钱包服务未配置时**直接跳过退款**且状态改成已取消
   （单没了、钱没退）。现在这两种情况返回 `ErrRefundUnavailable` 并整体回滚取消动作。
4. `membership` 域有**一模一样**的吞错补偿（旧 `app/internal/membership/service.go:226-235`，
   `_, _ = u.walletSvc.Credit(..., ":rollback")`）：该包不在本片白名单，**未动**，已登记为后续任务
   （改法就是复用 `txdb`）。因此 epic 那条「全仓 `git grep _, _ = u.walletSvc.Credit` 为 0」
   只在 mall/wallet/points 三包内成立，不隐瞒。
   → **同日已解除**：见 [`2026-11-04-membership-money-tx-followup.md`](./2026-11-04-membership-money-tx-followup.md)，
   两处补偿删除、全仓 `git grep` 仅剩一处测试自用例。

## 事务边界（人工核对 + 列出，验收第 5 条要求）

| 链路 | 事务内做的事 | 事务外做的事 |
|------|--------------|--------------|
| `Service.PayOrder` | `LockOrderForSettle`（订单行 `FOR UPDATE`）→ 积分 `Debit` → 钱包 `Debit` → `PayOrderLocal`（库存条件扣减 / 兑换码 `SKIP LOCKED` 占用 / 写 `wys_mall_payment` / 改单状态 / 写 `wys_mall_order_log` / 写 `wys_mall_audit_log`） | 入参校验、订单预读（归属/过期快路径）、渠道合法性 |
| `Service.CancelOrder`（已支付分支） | `LockOrderForSettle` → `CancelPaidOrder`（改状态 + 日志）→ 钱包 `Credit` 退款 → 积分 `Credit` 退款 | 订单预读、未支付分支走原 `CancelUnpaidOrder`（无资金动作，不进事务） |

- 事务体内**零**外部 HTTP / ES / MQ / 推送调用：`grep -E 'http|elastic|queue|publish|Push|jpush|rongcloud'`
  在 `repo_postgres.go:284-480`（`PayOrderLocal` 体）内无命中，`wallet`/`points` 的 `Credit/Debit`
  是纯 SQL（各自 `SELECT ... FOR UPDATE` + `UPDATE` + `INSERT ledger`）。
- **锁序**统一为「先订单行、后钱包/积分行」，支付与取消同序，避免两条链反向加锁死锁；
  嵌套的仓储 `Transaction()` 落在 GORM 的 SAVEPOINT 上（日志可见 `ROLLBACK TO SAVEPOINT sp…`），
  内层失败只回滚到保存点、外层错误仍然整体回滚。

## 代码改动

| 文件 | 改动 |
|------|------|
| `app/internal/txdb/txdb.go`（新） | `Begin` / `With` / `DB(ctx, def)`，注释写明「为什么走 context 而不是加参数」 |
| `app/internal/txdb/txdb_test.go`（新） | 3 个用例：无事务回落默认句柄、嵌套取最内层、nil 句柄被忽略 |
| `app/internal/mall/repository.go` | 接口加 `WithTx` + `LockOrderForSettle`；新错误 `ErrRefundUnavailable` |
| `app/internal/mall/repo_postgres.go` | `PayOrderLocal` / `CancelUnpaidOrder` / `CancelPaidOrder` / `LockOrderForSettle` 改用 `txdb.DB(ctx, r.db)`；实现两个新方法 |
| `app/internal/mall/service.go` | 支付/取消进 `WithTx`；删 `restorePayDebits`；已结算跳过扣款；退款错误不再吞 |
| `app/internal/wallet/repo_postgres.go` | `Credit` / `Debit` / `GetBalanceFen` 支持 ctx 事务 |
| `app/internal/points/repo_postgres.go` | `Credit` / `Debit` / `GetBalance` 支持 ctx 事务 |
| `app/internal/mall/{memwallet,service_balance_pay}_test.go` | 内存夹具补 `WithTx`（快照/还原模拟原子性）与 `LockOrderForSettle` |
| `app/internal/mall/service_money_tx_integration_test.go`（新） | 真库三条用例，`//go:build integration` |

未动：`app/cmd/api/main.go`（黑名单）—— 所以 `WithTx` 挂在**仓储接口**上而不是构造函数注入，
DB 句柄本来就只在水仓储里。`mall.CreateOrder`（`:213`）不在资金路径，保持原样（推广是 P2-9）。

## 真库用例（连接方式与清理策略，验收第 1 条要求）

```bash
docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p03_it"
GOPROXY=https://goproxy.cn,direct DATABASE_PASSWORD=postgres \
  go -C app test -tags=integration ./internal/mall/ -run TestMoneyTx -count=1 -v
```

- 连接：本机 Docker PostgreSQL 16（`my_go_study_postgres`），经 `core/database.NewPostgres` +
  `common/config.DatabaseConfig` 构造，库名走 env `P03_PG_DBNAME`（默认 `p03_it`）。
  **独立 scratch 库**，绝不连 `my_go_study` 业务库。
- 建表：用例内 `AutoMigrate` 11 张（钱包/积分账与流水、SKU、订单、订单行、支付、状态日志、兑换码、审计）。
  这里验的是事务边界，不是迁移保真度（后者由 P0-1 闸门覆盖）—— 口径写明白，不冒充「迁移跑通了」。
- 清理：`t.Cleanup` 逆序 `DropTable`；跑完 `p03_it` 回到空库。
- 连不上库时 `t.Skipf` 并报出库名，避免 `make test` 在无机配环境误红。

```
--- PASS: TestMoneyTxRollbackOnOrderFailure (0.23s)     # 库存不足→改单失败：余额 1000 不变、现金流水 0 条、订单仍未支付
--- PASS: TestMoneyTxConcurrentPayDebtsOnce (0.19s)     # 两并发支付：余额 700（只扣 300 一次）、流水 1 条、库存 5→4
--- PASS: TestMoneyTxCancelRefundsAtomically (0.25s)    # 取消退款回账；退款服务缺失时取消整体失败（状态仍为已支付）
ok  app/internal/mall  1.272s
```

## 其余验收

- `go -C app test ./internal/mall/... ./internal/wallet/... ./internal/points/... ./internal/txdb/... -count=1` → 四个包全 `ok`
- `go -C app test ./... -count=1` → 全绿（无 FAIL）
- `go -C app build ./...` / `go -C app vet -tags=integration ./internal/mall/` → 通过
- `git grep -E '_, _ = .*(Credit|Debit|Refund)' -- app/internal/mall app/internal/wallet app/internal/points` → **无命中**
  （当时 membership 两处命中，见 note 4；同日 follow-up 已清零）
- 新二进制对全迁移库启动冒烟：`/health=200`、`schema 版本校验通过 {version: 20260924110000, migrations: 25}`、
  DDL 关键字 0（顺带复验 P0-1 未被本片破坏）

## 已知残留

| 残留 | 状态 | 说明 |
|------|------|------|
| `membership` 域同款吞错补偿 | **已解除（2026-11-04 同日 follow-up）** | 复用 `txdb` 落地，改法与本片中 `PayOrder` 同形；见 [`2026-11-04-membership-money-tx-followup.md`](./2026-11-04-membership-money-tx-followup.md) |
| HTTP「下单→支付→取消」curl 冒烟 | **Deferred** | 需要登录态 + 商城目录种子 + 钱包余额（`p01_blank` 有 37 个 SKU 但 0 用户）。本片未改 handler/router，链路等价性由真库 service 层用例覆盖；HTTP 侧冒烟随 **P0-2**（支付回调，必然要走 HTTP）一起做并写进那份额度记录 |
| 内存夹具的「回滚」是快照/还原模拟 | **Degraded** + note | `payOrderRepo.WithTx` 用 `memCashRepo.snapshot/restore` 假演原子性，真实语义由 integration 用例承担；夹具只保证「编排走没走事务」这一层 |
| 支付幂等仍无 ledger 唯一约束 | **Deferred** | 现靠「先锁订单行」串行化并发支付；`wys_cash_ledger` 没有 `(user_id, reason, ref_id)` 唯一索引，属 P2 索引治理范围，本片不擅自加约束（会挡住合法的多次同因流水） |
