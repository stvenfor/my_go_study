# Slice — harden-h9-mall-mixed-payment-deduction（口径②：兑换率折算）

> ⚠ **Brief 原件落位说明（coordinator，2026-11-06 17:3x）—— 这是编排侧的缺陷，不是 worker 的**：本文件当时只是主仓的**未跟踪文件**（`git status` 里的 `?? plans/slices/harden-h9-…md`），从没进过 `e288553`，所以从 main 新开的 worktree 里**根本不存在**。worker 如实报告「没读过、也不声称读过」，并反过来按白名单重建了一份只含 Context Card 的占位文件、明确标注「非人批 Brief 原件」——这个处理是本轮 `no-silent-accept` 的正面样板。
> 处置：下面正文即**人批口径②的正式 Brief**，worker 的 Context Card 已完整保留在文末；两份路径现已内容一致。
> **防再犯**：派发前必须确认 Brief 在落点可见（`git ls-files` 里有它，或先把规划文档提交；本次已改为「主仓写 → 立即 `cp` 进 worktree」）。同类风险仍在：`plans/slices/harden-h10-*.md` 也未跟踪，派 h10 前必须同样处理。

> **状态：规则已批（2026-11-06 人批口径②），可动工。**
> 批下来的规则：**兑换率折算** —— `price` 语义重写为「**抵扣后应付现金**」，`price_points` 是「抵扣所需积分」，二者按兑换率折算后相加才是展示用原价。
> 人给的例子是「如 100 积分 = 1 元」，是**举例不是定值**；本 Brief 采纳 **100 积分 = 1 元** 为默认（`1 积分 = 1 分`，整除无余数），改动只有一行常量，人若换率只改常量 + 重跑存量折算。
> **口径落为决策记录**：`docs/adr/0019-mall-mixed-payment-exchange-rate.md`（**部分取代 ADR-0008** —— 结算规则照抄保留，被取代的只是 0008 对「points offset N yuan」在**定价/展示模型**上的否决）。

## 口径② 对缺陷定性的更正（重要，勿按旧 Brief 写码）

旧 Brief 把现行实现定性为「同一笔价款既收现金又扣积分 = 双重计费」。在口径②下这个定性**不成立**：

| 证据 | 内容 |
|------|------|
| `model/entity/mall.go:211` | `Price` / `PricePoints` 是**定价输入**，不是「原价的两份表达」；② 下 `Price` 本来就该写「抵扣后应付」 |
| `app/internal/mall/service.go:347-348` | `total += unit × qty` 累加的是应付现金；`:349` `linePts = PricePoints × qty` 累加的是应付积分 —— 两条独立累加在 ② 下**正是正确写法** |
| `app/internal/mall/service.go:672-673 / :699 / :705-712` | 收全额 `order.Amount` + 扣全额 `order.TotalPoints` = 收「应付现金」+ 扣「应付积分」，**没有二次收费** |
| 只读 SQL 实跑（coordinator，2026-11-06，业务库 `my_go_study`） | Mixed SKU **1 行**（`sku_id=37`，`price=9.90`，`price_points=50`）；Mixed 订单 **0 行**；Cash-only 33 行 / Points-only 1 行 / SKU 共 37 行 |

→ 真正的风险不在代码算式，而在**口径没有唯一定义处**：`docs/` 58 篇里 **0 篇商城文档**，`CreateSKUInput`（`service.go:84-94`）对店员录入零口径提示，兑换率在代码里根本不存在。店员把「原价」直接填进 `price` 时，那一行 SKU 就同时收原价现金 + 全额积分 —— **这才是双重计费唯一会真实发生的形态，且它是数据问题，改支付路径治不了。**
`sku_id=37` 属于「已按 ② 定价」还是「把原价误填进 price」，从库里判不出来 → 需人裁定（见下 ④）。

## Slice Brief

- SOURCE_MODULE: `model/entity/mall.go`（定价字段与派生函数所在）、`app/internal/mall`（录入与结算）
- TARGET_MODULE: 同一批文件 + 新增「兑换率单一来源」与口径文档
- Source entry: `service.go:97-132`（`CreateSKU` 录入校验）、`service.go:660-730`（`Pay`，**只读参考，本轮不改**）、`service.go:769-773`（退款退积分）、`entity/mall.go:88-101`（`MallSKUPaymentMode`）
- Target entry: `entity/mall.go` 新增 `MallPointsPerYuan` 常量 + `PointsValueFen(points int64) int64` 折算函数
- **本轮 ONLY**（4 项，全做完才算可验收）:
  1. **兑换率唯一来源**：`model/entity/mall.go` 加 `MallPointsPerYuan int64 = 100` 与 `PointsValueFen`。要求：整除路径无余数；若将来率不整除 100，函数内必须显式取整方向并注释「余数归商家还是归顾客」——写进函数注释，不许散在各调用点各自折算。
  2. **口径落文档**：新建 `docs/mall-mixed-payment.md`：① 两个字段各自语义（`price` = 抵扣后应付现金，非原价）② 算式 `原价(分) = price 的分数 + PointsValueFen(price_points)` ③ 真实算例（sku 37：应付 ¥9.90 + 50 积分 ⇒ 折合原价 ¥10.40）④ 店员录入约定（先定原价与想抵的积分，再反填 `price = 原价 − 抵扣价值`）⑤ 改兑换率必须重刷存量价的风险声明 ⑥ 与 `payment_mode` 派生关系。
  3. **不变量测试**（`app/internal/mall/service_test.go` 或新 `_test.go`）：Mixed 单实付现金（分）== `price` 分数 × qty；实扣积分 == `price_points` × qty；`实付现金 + PointsValueFen(实扣积分) == 展示原价`；退款对称（`:773` 退积分 == 实扣，现金退款 == 实付），**且不存在任何路径让同一分既收现金又扣积分**。
  4. **存量一行裁定**：默认**不动数据**。若人判 `9.90` 是误填的原价，则出一条 `migrations/` 把 `sku_id=37` 的 `price` 改为 `9.40`（= 9.90 − 0.50），`.down.sql` 还原为 `9.90`；若人判它已符合 ②，则本项只写进文档、零迁移。**未经人裁定不得改这行数据。**
- 不做:
  - **不改 `service.go` 支付/结算主路径**（`:320-390`、`:660-730`）——② 下它已符合口径；任何「先算抵扣再收现金」的改造都会造成二次打折，属反向资损
  - 不动 `MallSKUPaymentMode`（`entity/mall.go:88-101`）派生逻辑
  - 不新增 `deduct_ratio` / `max_deduct_points` 列（那是**未批**的规则③）
  - 兑换率**不进配置文件**：进配置会让存量 `price` 语义随运行时漂移，属资损面
  - 不动 `transactions` 单位（h6b 锁区）、不动 `schemaboot` / 唯一索引名（h5 / h10 锁区）
  - 不顺手给 `normalizeMoney`（`service.go:816`）加校验（h6 残留第 5 条，另开 Slice）
  - 不改任何 JSON 字段名；若要前端展示「折合原价」，先出跨仓 Brief（`docs/coding-playbook-dual-repo.md`）
- 验收:
  - [ ] `PointsValueFen` 是仓库内**唯一**积分→现金折算入口（`grep` 证明没有第二处手写 `/100`、`*100`）
  - [ ] `docs/mall-mixed-payment.md` 存在且含上述 6 点；算例数字与库内真实行一致
  - [ ] 不变量测试覆盖「现金分 + 积分折算分 == 原价分」，余数恒为 0（在 100/元 下）并有测试证明
  - [ ] 退款对称测试存在且通过：退现金 == 实付现金、退积分 == 实扣积分
  - [ ] 存量处置有据：要么「零迁移 + 人已裁定」的书面记录，要么迁移文件 + `make migrate-up`/`migrate-down` **实跑输出**（`.down.sql` 必须真能还原；勿再靠 `auto_migrate` 成功与否或 health 绿判断，h10 已证明 42704 会被 `BestEffort` 吞掉）
  - [ ] `docs/placeholder-inventory.md` 追加本轮残留（如「店员录入无口径校验，仅靠文档约束」「兑换率仍是常量、改率需人工重刷」）
  - [ ] 状态词只允许 Migrated / Degraded / Placeholder / Deferred / Partial；本轮只改后端 → **Partial**
  - [ ] **与 ADR-0008 的关系必须写准**（本轮验收新增，我已复核过一处偏差）：不许写成「② 正是 ADR-0008 所规定的」——0008 白纸黑字 **Rejected: "points offset N yuan"**，而 ② 接受的恰恰是这个模型（只是接受在**定价/展示层**、拒绝仍在**结算层**）。正确表述见 `docs/adr/0019-…`：结算两腿并列互不净额 = 继承 0008；兑换率折算与展示原价 = 取代 0008 的那条 Rejected
  - [ ] **必须明写「本轮之后顾客可见价格零变化」**：后端没在 offer/detail 出参加「折合原价 / 已抵 ¥x」，所以买家侧数字与改口径前完全一致；要让抵扣可感知需跨仓 Brief。这条不许被「已落地口径」的说法盖过去
- 文件白名单:
  - `model/entity/mall.go`（新增常量与折算函数）
  - `app/internal/mall/service.go`（**仅限**调用折算函数/注释口径；不得改结算算式）、`app/internal/mall/service_test.go`、`app/internal/mall/*_test.go`
  - `docs/mall-mixed-payment.md`（新建）
  - `migrations/`（仅当人裁定要改 sku 37；必须 `make migrate-up/down` 实跑）
  - `docs/acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md`
  - `docs/placeholder-inventory.md`（追加式）
  - `plans/slices/harden-h9-mall-mixed-payment-deduction.md`（仅 Context Card）
- 文件黑名单:
  - `app/cmd/api/main.go`、`app/internal/delivery/http/router/**`
  - `docs/adr/**`（0019 是 coordinator 代笔的人批决策记录，只读引用，不许改）
  - `app/internal/transaction/**`、`app/internal/payment/**`（h6b / h8 锁区）
  - `model/entity/im.go`、`auth_user.go`、`schemaboot*`（h10 / h5 锁区）
  - `app/go.mod`、`app/go.sum`（h2 锁区）
  - Flutter 仓（跨仓另开 Brief）
- 验证命令:
  - `go -C app test ./internal/mall/ -run 'Mixed|Deduction|PointsValue|Refund' -count=1`
  - `go build all`
  - 唯一折算入口：`grep -rn "PointsValueFen\|/ *100\|100 *积分" app/internal/mall model/entity/mall.go`
  - 存量复核（只读，业务库允许 SELECT）：`docker exec my_go_study_postgres psql -U postgres -d my_go_study -Atc "select sku_id,price,price_points from wys_mall_sku where price::numeric>0 and price_points>0;"`
  - 集成测试（若写）连 scratch 库 `p09_it`（`P03_PG_DBNAME`），**绝不连业务库 `my_go_study`**
  - 若出迁移：`make migrate-up` + `make migrate-down` 真实输出贴进证据
- 证据: `docs/acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md`

## 遗留决策（不阻塞动工，但验收时要人签字）

1. 兑换率取值 100 积分 = 1 元 是本 Brief 采纳的默认，人可换值；换值即触发 sku 存量重刷。
2. `sku_id=37` 是「已按 ② 定价」还是「原价误填进 price」——只有商家知道，coordinator 与 worker 都无权替它定性。

## Context Card — harden-h9-mall-mixed-payment-deduction（worker 原文保留）

- 已完成（本轮后端子项，**不构成整片完成**；整片 status=`Degraded`）:
  - `model/entity/mall.go`：`MallPointsPerYuan=100` + `PointsValueFen`（全仓唯一积分→现金折算入口，注释写清取整方向 / 余数归属）。
  - `app/internal/mall/service.go`：**仅注释口径**；支付 / 结算 / 退款主路径零改动（`git diff` 为纯新增行，无删除）。
  - `app/internal/mall/service_mixed_payment_test.go`：实付现金 / 实扣积分 / 展示原价 / 退款对称 / 无双重计费五类不变量，单测全绿。
  - `docs/mall-mixed-payment.md`：两字段语义、原价算式、`sku_id=37` 算例、店员录入约定、改率风险、与 `payment_mode` 的派生关系。
  - 本文件 Context Card、`docs/acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md`、
    `docs/placeholder-inventory.md`（追加式）。
- 未做 / Deferred:
  - **跨仓另一半未做**：Flutter 商城页「展示原价」与店员录入端的口径语义校验本轮未碰（Flutter 仓在黑名单）→ 整片 `Degraded`，不自称完成。
  - 存量 `sku_id=37` **零迁移、零 UPDATE**：属「已按口径②正确定价」还是「把原价误填进 price」只有商家能定性；
    执行方与人（本轮）均无权定性，作为待人裁定项写进 acceptance-record 残留。
  - `CreateSKUInput` 未加「原价 / 抵扣」语义校验（超出本轮 ONLY，靠文档 + 代码注释约束）。
  - 调整 `MallPointsPerYuan` 须重刷存量价（docs §7）；本轮未改率。
  - 集成测试（`-tags=integration`）本轮未运行：不变量测试是纯单测，不连任何库；业务库 `my_go_study` 仅 1 次只读 `SELECT`。
- 关键文件:
  - `model/entity/mall.go`、`app/internal/mall/service.go`、`app/internal/mall/service_mixed_payment_test.go`
  - `docs/mall-mixed-payment.md`、`docs/acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md`
- harness: post ok? — 核对命令（结果见 acceptance-record「验证证据」段）：
  - `go -C app test ./internal/mall/ -run 'Mixed|Deduction|PointsValue|Refund' -count=1`
  - `go build all`
  - `grep` 证明全仓无第二处手写积分→现金 `/100` 折算
  - 业务库只读:`docker exec my_go_study_postgres psql -U postgres -d my_go_study -Atc "select sku_id,price,price_points from wys_mall_sku where price::numeric>0 and price_points>0;"`
- 下一 Slice 建议:
  - 待人补正式「Slice Brief」正文并裁定 `sku_id=37`；若裁定为「原价误填」，另立数据变更 Slice、由人执行重刷（程序不代定）。
  - 跨仓另一半：Flutter 展示「原价 = price + 积分折现」+ 店员录入端口径校验。
  - 若将来调整 `MallPointsPerYuan`，按 docs §7 重刷存量价并回归不变量测试。
- 已知坑:
  - **不要改支付 / 结算主路径**：口径②下「收全额 `order.Amount` + 扣全额 `order.TotalPoints`」就是正确写法；
    改成「先算抵扣再收现金」= 二次打折 / 反向资损，直接判不合格。退款侧按 `row.TotalPoints` 全额退同理。
  - `price` 语义是「抵扣后应付现金」，**不是原价**；把原价填进 `price` 才会造成双重计费（真实风险在录入端，不在结算端）。
  - 状态词只用五态（`Migrated/Degraded/Placeholder/Deferred/Removed`）；`Partial` 不在其列。
    本轮只改后端 → `Degraded` + note「跨仓另一半未做」。
  - 业务库 `my_go_study` 只允许 `SELECT`；集成测试只连 scratch 库 `p09_it`（`P03_PG_DBNAME` 指过去），HTTP 用 8081。
  - 不 commit、不 push、不改 git config；白名单外的锁区（h6b transaction / h10 im / h2 go.mod）绝不碰。

### coordinator 验收补记（2026-11-06 17:4x）
- 我独立复跑：`go -C app test ./internal/mall/ -run 'Mixed|Deduction|PointsValue|Refund' -count=1` → **ok 0.844s**；`git diff --numstat` = service.go **17/0**、entity/mall.go **25/0**、placeholder-inventory **31/0**（纯新增，主路径零删除）；只读 SQL 复核 `sku_id=37` 仍 `9.90 / 50`、`updated_at=2026-10-21` → **零 UPDATE 承诺成立**。
- `grep PointsValueFen` 实测：**生产代码零调用点**（只有常量/函数定义、注释、测试）—— 这是 ② 的正确结果（结算不需要折算），但也证明「顾客可见零变化」这条验收项必须补，已列为待改。
- 待改两处（小范围，不推翻交付）：① `docs/mall-mixed-payment.md` 把 ② 说成「这正是 ADR-0008 的规定」不准（0008 明确 Rejected "points offset N yuan"），改为按 ADR-0019 的「继承结算规则 / 取代其 Rejected」表述；② 文档与 record 各加一条「本轮买家侧价格零变化，展示折合原价需跨仓 Brief」。
