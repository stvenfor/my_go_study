# Acceptance Record — harden-h9-mall-mixed-payment-deduction

- 日期：2026-11-06
- TASK：`task_eeb8ddbf228a` / dispatch `ctx_9511fbba0bcc`
- Slice：`plans/slices/harden-h9-mall-mixed-payment-deduction.md`
- worktree：`/Users/qpy/orca/workspaces/my_go_study/harden-h9`（分支 `maoqian/harden-h9`，HEAD `e288553`）
- 口径：人批「口径②」—— `price` = 抵扣后应付现金，`price_points` = 抵扣所需积分，100 积分 = 1 元（1 积分 = 1 分）
- 主交付物：全仓唯一折算入口 `entity.PointsValueFen` + 口径文档 + 不变量测试 + 残留登记

## status: **Degraded**

note：

1. 本轮**只改后端**（Go）四项；**跨仓另一半未做**（Flutter 商城页「展示原价」与店员录入端的口径语义校验）。
   按 [`.harness/rules/status-enums.md`](../../.harness/rules/status-enums.md) 五态取 `Degraded`，
   **不自称完成**、不标 `Migrated`（`Partial` 不在五态内）。
2. 口径②下「收全额 `order.Amount` + 扣全额 `order.TotalPoints`」在
   `app/internal/mall/service.go`（`CreateOrder` / `settleLocal` / `SettleChannelPaid` / `CancelOrder`）
   就是正确写法 —— **支付 / 结算 / 退款主路径零改动**（`git diff --numstat` = `17 0` / `25 0`，纯新增注释、零删除行）。
   谁改成「先算抵扣再收现金」= 二次打折 / 反向资损，直接判不合格；退款侧按 `row.TotalPoints` 全额退同理是对的。
3. 真实风险在**录入端**：`CreateSKUInput` 对店员录入零口径约束，店员把原价填进 `price` 时那一行才变成双重计费。
   本轮只补文档 + 代码注释，未改校验逻辑（超出本轮 ONLY）。
4. 存量 `sku_id=37` **默认零迁移、不动数据**：见「已知残留」第 1 条。
5. **本轮之后买家可见价格零变化**（硬事实，Brief 验收新增项，见 §验证证据 3b 与残留）：`grep` 实测
   `PointsValueFen` 在**生产代码零调用点**（只有定义 / 常量 / 注释 / 测试）—— 这是口径②的正确结果
   （结算不需要折算），但也意味着本轮**没有任何接口输出「折合原价 / 已抵 ¥0.50」**，
   offer / detail 出参未新增字段，买家侧数字与改口径前**完全一致**。要让抵扣对顾客可感知，
   需跨仓 Brief 改 `MallSKUOffer` / `MallProductDetail` 出参（加字段 = 改 JSON 契约）。
   **不得**用「口径已落地 / 折算入口已补」盖过去。

## 本轮四项（对照 dispatch ONLY）

| # | 交付 | 落点 |
|---|------|------|
| 1 | `MallPointsPerYuan int64 = 100` + `PointsValueFen(points int64) int64`（注释写清取整方向 / 余数归属） | `model/entity/mall.go:54,119` |
| 2 | 两字段语义、原价算式、`sku_id=37` 算例、店员录入约定、改率须重刷存量价的风险声明、与 `payment_mode` 的派生关系 | `docs/mall-mixed-payment.md` |
| 3 | 实付现金 / 实扣积分 / 展示原价 / 退款对称 / 无双重计费五类不变量 | `app/internal/mall/service_mixed_payment_test.go` |
| 4 | 存量 `sku_id=37` 零迁移、待人裁定项写入残留 | 本记录「已知残留」 |

`app/internal/mall/service.go` 仅按白名单「注释口径」补 6 处注释，无任何逻辑改动。

## 代码改动（白名单内）

| 文件 | 改动 |
|------|------|
| `model/entity/mall.go` | 新增常量 + 折算函数（+25 行，0 删除） |
| `app/internal/mall/service.go` | +17 行注释（`CreateSKUInput` / `CreateOrder` / `settleLocal` / `CancelOrder` 口径），0 删除行，逻辑零改动 |
| `app/internal/mall/service_mixed_payment_test.go`（新） | 内存夹具 + 4 个测试函数 |
| `docs/mall-mixed-payment.md`（新） | 口径权威文档 |
| `plans/slices/harden-h9-mall-mixed-payment-deduction.md`（新） | Brief Context Card（原件缺失，重建；见残留 2） |
| `docs/acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md`（本文件，新） | 验收记录 |
| `docs/placeholder-inventory.md` | 追加式登记（条目 22） |

## 不变量（逐条对应 dispatch 第 3 项）

| 不变量 | 断言位置 | 结果 |
|--------|----------|------|
| 实付现金额 == `price` 分数 × `qty` | `TestMixedPaymentDeductionInvariants` + `TestMixedPaymentRefundSymmetry`（真实钱包流水） | PASS |
| 实扣积分 == `price_points` × `qty` | 同上 | PASS |
| 实付现金分 + `PointsValueFen(实扣积分)` == 展示原价分 | `TestMixedPaymentDeductionInvariants` + `TestMixedPaymentRefundSymmetry` | PASS |
| 退款对称（退现金 == 实付、退积分 == 实扣） | `TestMixedPaymentRefundSymmetry` | PASS |
| 不存在同一路径既收现金又扣同一份价值 | `TestMixedPaymentDeductionNoDoubleCount`：两腿恰好切分展示原价、现金腿严格小于展示原价 | PASS |
| `sku_id=37`：990 分 + 50 分 = 1040 分 = 10.40 元 | `TestPointsValueFen` + `TestMixedPaymentDeductionInvariants` | PASS |

## 验证证据（2026-11-06 实测）

```bash
# 1) 定向单测
go -C app test ./internal/mall/ -run 'Mixed|Deduction|PointsValue|Refund' -count=1
# → ok  	app/internal/mall	0.566s
#   -v 复跑：TestPointsValueFen / TestMixedPaymentDeductionInvariants /
#     TestMixedPaymentRefundSymmetry / TestMixedPaymentDeductionNoDoubleCount 全 PASS，
#     既有 TestMallBalancePayAndCancelRefund 亦 PASS。

# 2) 全工作区构建
go build all
# → go build all: OK

# 3) 唯一折算入口 / 无第二处手写 /100 折算
grep -rn "PointsValueFen\|MallPointsPerYuan" --include='*.go' .
# → 唯一定义 model/entity/mall.go:119；常量 :54；service.go 中 6 处命中全是注释，
#   无生产调用、无净额。
grep -rnE '[^/]/[[:space:]]*100([^0-9]|$)|[^/]/100\.0' --include='*.go' .
# → 11 条命中，逐条性质：
#   model/entity/cash_money.go:30、app/internal/payment/service.go:285 = 分→元格式化；
#   app/internal/purchase_calculator/service.go:198,212,214,218,221,259,263,281 = BPS→百分比 / round2；
#   app/internal/usecase/analytics_usecase.go:291 = round2。
#   其中含 points/积分 的命中：无（`grep ... | grep -iE 'points|积分'` → 无命中）。
# → 证明全仓无第二处手写积分→现金折算。

# 3b) 生产代码零调用点 + 无第二处手写 *100 折算（Brief 验收第 46 / 54 行）
grep -rn "PointsValueFen" --include='*.go' app/internal/mall model/entity/mall.go | grep -v '_test.go' | grep -vE ':[0-9]+:[[:space:]]*//'
# → 仅 model/entity/mall.go:119 的函数定义；生产调用点 0（结算不需要折算 = ② 的正确结果）。
grep -rnE '\* *100' --include='*.go' app/internal/mall model/entity/mall.go
# → 命中：integration 测试的 SKU id 算术（orderID*100+1）、mall.go:115 的溢出说明注释、
#   mall.go:125 的唯一折算入口 yuan*100 + rem*100/MallPointsPerYuan —— 无第二处手写积分→现金折算。
# → 推论：本轮没有任何接口输出「折合原价 / 已抵 ¥0.50」，买家可见价格零变化（见 note 5 与残留）。

# 4) 存量只读 SQL（业务库 my_go_study，仅 SELECT，零 DDL/DML）
docker exec my_go_study_postgres psql -U postgres -d my_go_study -Atc \
  "select sku_id,price,price_points from wys_mall_sku where price::numeric>0 and price_points>0;"
# → 37|9.90|50   （唯一命中行；本轮未 UPDATE / 未迁移）
```

- 集成测试（`-tags=integration`）本轮**未运行**：不变量测试是纯单测，不连任何库；未触碰 scratch 库 `p09_it`，
  业务库 `my_go_study` 全程只有上面 1 次只读 `SELECT`。dispatch 约束「集成测试只连 scratch 库 `p09_it`
  （`P03_PG_DBNAME` 指过去）、HTTP 用 8081、绝不连业务库写」在本轮无写操作，天然满足。
- `git status --short`：7 个白名单文件（3 改 + 4 新）；黑名单零改动
  （`app/cmd/api/main.go` / `router/**` / `app/internal/transaction/**` / `app/internal/payment/**` /
  `model/entity/im.go` / `auth_user.go` / `schemaboot` / `app/go.mod` / `go.sum` / `go.work` / Flutter 仓 /
  任何 `.env*` 全部未碰）。
- `go.work` / `go.work.sum` / `thunder/logs/logs.go` / `bin/migrate` 为 dispatch 预先复制且被 `.gitignore` 挡住的文件，
  非本轮改动（开工 `git status --short` 为空可证）。
- 不 commit、不 push、不改 git config；无 `rm -rf` / `git branch -D` / `git worktree remove --force`。

## 已知残留

| 残留 | 状态 | 说明 |
|------|------|------|
| **存量 `sku_id=37` 待人裁定** | **Deferred** | 只读 SQL 显示 `price=9.90`、`price_points=50`。它属于「已按口径②正确定价」还是「把原价误填进 `price`」**只有商家知道**；worker 与人（本轮）均无权定性。本轮**默认零迁移、不动数据**，未 `UPDATE` 该行（自行 UPDATE = 不合格）。建议：商家核对原始定价单后裁定；若需重刷，另立数据变更 Slice 由人执行，程序不代定。 |
| **权威 Brief 原件缺失** | **Deferred** | `plans/slices/harden-h9-mall-mixed-payment-deduction.md` 在 HEAD `e288553` 不存在（worktree / git 跟踪 / git 历史 / 主仓均无；`.harness/changes/current.md` 只列 H1–H8）。本轮依据 dispatch 权威消息（口径②已批，明确覆盖注入 preamble 旧描述）+ `docs/adr/0008-mall-points-and-mixed-payment.md` 落地；已按白名单重建一份**仅含 Context Card**、明确标注「非人批原件」的文件，待人以正式 Brief 覆盖其「Slice Brief」正文。执行方**未声称读过不存在的 Brief**，也未虚构人批正文。 |
| **买家侧展示「折合原价 / 已抵」未做（顾客可见价格零变化）** | **Deferred** | 硬事实：本轮 `PointsValueFen` 在生产代码零调用点，offer / detail 出参未新增任何字段，买家看到的数字与改口径前**完全一致**；没有任何接口输出「折合原价 / 已抵 ¥0.50」。要让抵扣对顾客可感知，必须另出跨仓 Brief 改 `MallSKUOffer` / `MallProductDetail` 出参（加字段 = 改 JSON 契约）+ Flutter 商城页展示 + 店员录入端语义校验。本条不许被「口径已落地」盖过去，是整片 `Degraded` 的成因之一（见 note 5、docs §9）。 |
| 跨仓另一半（Flutter 展示原价 / 店员录入端语义校验） | **Degraded 的成因** | 本轮只改后端，整片 `Degraded`，不自称完成。 |
| `CreateSKUInput` 无「原价 / 抵扣」语义校验 | **Deferred** | 只校验金额格式与非负；本轮仅补文档与代码注释，未改校验逻辑（超出本轮 ONLY）。 |
| 改率须重刷存量价 | **Deferred** | `MallPointsPerYuan` 为编译期常量；改动会改变所有展示原价派生值，见 `docs/mall-mixed-payment.md` §7。本轮未改率。 |
| 集成测试 / HTTP 冒烟 | **Deferred** | 见上；本轮纯单测，不连库、不发 HTTP。 |
