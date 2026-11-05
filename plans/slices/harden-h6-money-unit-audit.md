# Slice — harden-h6-money-unit-audit

## Slice Brief
- SOURCE_MODULE: `model/entity`（金额相关实体）、`app/internal/{transaction,used_car,auto_finance,mall,wallet,payment}`
- TARGET_MODULE: 一份落盘的**口径与迁移方案**文档（本 Slice 不改代码）
- Source entry:
  - 整数分派（正确一侧）：`wallet`/`points`/`payment`/`mall` 的 `*Fen` 字段与 `Entity.CashReason*` 流水
  - `float64` 金额派（风险一侧，实测行号）：`model/entity/transaction.go:16,28,36`、`model/entity/transaction_record.go:14`、`model/entity/used_car_order.go:138,180`、`model/entity/auto_finance.go:52,60-70`（其中 `:37-45` 是**百分比/利率**，性质不同，必须分类）
  - 客户端直收：`app/internal/used_car/handler.go:94`（`Amount float64` 来自 JSON body，无精度与上限校验）
  - 反例（不是钱，别误伤）：`model/entity/mall.go:162,177 CoverAspect`（图片宽高比）、`analytics_record.go`、`short_video.go`
- Target entry: `docs/money-unit-audit.md` —— 每个字段的「单位 / 是否参与资金 / 存储类型 / 客户端契约 / 迁移代价」+ 一份可执行的迁移与灰度方案
- 本轮 ONLY: **只读盘点 + 方案**（决策门 4：默认不落地）
- 不做: 不改任何实体、迁移、handler、DTO；不动 Flutter；不做「顺手加个校验」（要加就单独立 Slice）
- 验收:
  - [ ] 字段清单完备性**用排除法证明**：`grep -rn 'float64' model/entity/` 的每条都要么归入「资金/金额类」要么显式标为「非钱（比例/尺寸/统计）」并写明理由 —— 不接受「只列我想到的」
  - [ ] 每个金额字段标注：当前 DB 列类型（`numeric`/`float8`/`double precision`，查 `\d` 或 migrations）、是否已被 `wallet`/`payment` 事务链路引用、是否出现在 Flutter 契约
  - [ ] 明确列出**混算点**（同一笔钱在两套单位间来回的地方），没有也要给 grep 证明「不存在」
  - [ ] 迁移方案含：加列→回填→双写→切读→删列的具体 SQL 草案 + 回滚点；每步的停机/一致性风险；`transaction` 表在 PostgREST/RLS 与 ES 索引两侧的连带影响（`docs/elasticsearch-transactions.md`）
  - [ ] 精度事故评估：给出「现有数据里 float 是否已经失真」的可执行核查 SQL（例如 `amount <> round(amount, 2)`）
  - [ ] 方案落 `docs/money-unit-audit.md`，acceptance-record 状态写 **Partial**（只交付方案）或 **Deferred**（若人批后不落地）—— 不许写 Migrated
- 文件白名单:
  - `docs/money-unit-audit.md`（新建）
  - `docs/placeholder-inventory.md`（追加式：登记「金额两套口径」条目与状态）
  - `docs/acceptance-records/2026-11-05-harden-h6-money-unit-audit.md`
  - `plans/slices/harden-h6-money-unit-audit.md`（仅 Context Card）
- 文件黑名单:
  - `model/**`、`app/**`、`migrations/**`、`configs/**`
  - 任何 `_test.go`
- 验证命令:
  - `grep -rn 'float64' model/entity/ | wc -l`（与清单条数一致）
  - `git diff --stat` 证明本 Slice **零代码改动**（只允许 docs/ 变化）
  - `./scripts/load-env.sh psql -h 127.0.0.1 -U postgres -d <只读查询>` —— 只跑 `SELECT`，**禁止任何 DDL/DML**（业务库 `my_go_study` 属破坏性边界）
- 证据: `docs/acceptance-records/2026-11-05-harden-h6-money-unit-audit.md` + `docs/money-unit-audit.md`

## Context Card — harden-h6-money-unit-audit
- 已完成:
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
