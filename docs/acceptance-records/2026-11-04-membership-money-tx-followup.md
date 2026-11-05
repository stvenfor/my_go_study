# Acceptance Record — membership-money-tx-followup（P0-3 残留 Task #12）

- 日期：2026-11-04
- 来源残留：`docs/acceptance-records/2026-11-04-parity-p0-3-money-tx-atomic.md` note 4 + 「已知残留」表首行；
  `docs/placeholder-inventory.md` 条目 13 残余 / 条目 18
- 主交付物：membership 域资金写全部走 `txdb`，删掉两处 `_, _ = u.walletSvc.Credit(..., ":rollback")` 吞错补偿
- 关联：`plans/epics/xhs-parity-p0-correctness.md` §P0-3 / §P0-2

## status: **Migrated**（membership 资金事务化）＋ 1 条未写真库用例（见「诚实口径」）

## 改了什么

| 文件 | 改动 |
|------|------|
| `app/internal/membership/repository.go` | `Repository` 新增 `Transaction(ctx, fn)`，注释写明「为什么不能再靠 Credit 补偿」 |
| `app/internal/membership/repo_postgres.go` | 实现 `Transaction` → `txdb.Begin(ctx, r.db, fn)`；`MarkOrderPaid` / `ExtendEntitlement` 改用 `txdb.DB(ctx, r.db).WithContext(ctx)` |
| `app/internal/membership/service.go` | `completeBalanceBuyout`：扣款 + 改单 + 发权益并入一个事务，**删掉两处 `_, _ = Credit(":rollback")`**；`ConfirmBuyout` 渠道分支：`MarkOrderPaid` + `ExtendEntitlement` 并入同一事务，`ErrOrderPaid` 移到**事务外**再递归确认（事务里递归会白滚一遍） |
| `app/internal/membership/service_test.go` | 夹具实现 `Transaction`（快照/还原**假演**原子性，与 mall 夹具同构）；新增 `TestMembershipBalanceBuyoutRollsBackDebit`；三处余额用例绑定 `repo.cash` |
| `app/internal/membership/service_balance_tx_integration_test.go`（新） | 真库两条用例，`//go:build integration` |

## 事务边界

| 链路 | 事务内 | 事务外 |
|------|--------|--------|
| `completeBalanceBuyout` | 钱包 `Debit` → `MarkOrderPaid`（条件更新，`RowsAffected==0` 即 `ErrOrderPaid`）→ `ExtendEntitlement`（内层自带 `Transaction` → 落 **SAVEPOINT**） | 入参校验、订单创建、渠道预支付 |
| `ConfirmBuyout`（微信/支付宝） | `MarkOrderPaid` → `ExtendEntitlement` | **`pay.ConfirmRefPaid`（核销渠道流水）先于本事务且不在本事务内** —— 见「诚实口径」第 3 条 |

- 锁序：membership 路径是「先钱包行、后订单行」；mall 支付路径是「先订单行、后钱包行」。
  两域锁的是**不同订单表**（`wys_mall_order` / `wys_membership_order`），不构成锁环，无死锁路径。
- 事务体内零外部 HTTP / ES / MQ / 推送调用（沿用 P0-3 立的规矩）。

## 验证

```bash
go build all                                                  # 退出 0
go -C app test ./internal/membership/ -count=1                # ok
go -C app test ./... -count=1                                 # 无 FAIL
go -C app vet -tags=integration ./internal/membership/        # 通过
docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p03_it"
DATABASE_PASSWORD=postgres P03_PG_DBNAME=p03_it \
  go -C app test -tags=integration ./internal/membership/ -run TestMembershipBalanceTx -count=1 -v
```

```
--- PASS: TestMembershipBalanceTxRollsBackDebit        (0.08s)   # 单已付→再购买：余额 10000 不变、现金流水 0 条、权益 0 条
--- PASS: TestMembershipBalanceTxConcurrentDebtsOnce   (0.09s)   # 两并发：余额 7000（只扣一次 3000）、流水 1 条、权益 1 条
```

`git grep -E '_, _ = .*(Credit|Debit|Refund)' -- app` → 仅剩 `app/internal/wallet/service_test.go:249`（测试自身丢弃返回值，非生产补偿）。

## 诚实口径

1. **回滚用例不是空断言**：改动前同一场景会留下 **2 条现金流水**（扣款 + `:rollback` 补偿），
   现在断言 `wys_cash_ledger` 为 **0 条** —— 这条断言在旧实现下必然失败，所以它真的在测东西。
2. **内存夹具的回滚仍是快照/还原假演**（`memMembershipRepo.Transaction`），与 mall 夹具同级别的口径；
   真实语义由上面的 integration 用例承担。
3. `ConfirmBuyout` 的**渠道核销没并进来**：`pay.ConfirmRefPaid` 先把 `wys_payment_records` 置 paid
   （它自己的事务），随后才是 membership 的改单+发权益事务。两段之间进程被杀 → 流水 paid、单未付、
   权益未发。这不丢钱（款确实收到），可由查单任务/再次回跳收敛，但**跨域单事务**要等条目 18 的
   `RegisterSettler` 落地才算真正收口。
4. 因此本条 residual 只解掉「发权益未事务化」，**未解掉「无业务核销器」**：条目 18 保持 `Degraded`，
   P0-2 验收记录 F2 相应改写窄。
5. `ConfirmBuyout` 渠道路径的两段写**没单独写真库用例**：它复用与余额路径**同一个** `Transaction` 原语
   （该原语已由上面两条真库用例证明），单测 `TestMembershipConfirmIdempotent` 覆盖 `ErrOrderPaid` 递归分支。
   口径：同形复用 + 单测覆盖，非真库直测。

## 下一步

条目 18（`pay.RegisterSettler(PaymentRefMembershipOrder, memSvc)`）—— 仓储 `Transaction` 已就绪，
改法与 mall 的 `SettleChannelPaid` 同形，落地后回调即可主动发权益，且上面第 3 条的跨域窗口一并消失。
