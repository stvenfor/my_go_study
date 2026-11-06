# Current change (harness pointer)

| 字段 | 值 |
|------|-----|
| Active | **backend-hardening H1–H8 · Task DAG 已建，待派发** |
| Program | `plans/2026-11-05-backend-hardening-program.md` |
| Epic | `plans/epics/backend-hardening.md` |
| Slices | `plans/slices/harden-h1…h8.md` + `harden-h6b-transaction-money-unit.md`（共 **9** 份 Brief） |
| Orca Run | `run_2960cd5ac4ef` —— **只建 Task DAG，未 worker-start** |
| Role | **coordinator**：只做需求澄清 / 拆解 / 验收打回；代码由 worker 写 |
| 上一轮 | `cs-agent-infra` C1–C4（Partial，见下方历史指针）· `xhs-parity` P0-1/2/3/4/5/6/7 已验收 |

## Task DAG（`run_2960cd5ac4ef`，2026-11-06 建，deps 已机检确认）

| key | task_id | 标题 | deps | 状态 |
|-----|---------|------|------|------|
| h1 | `task_7520b7ab03de` | 仓库卫生：import 覆盖机检 + 二进制杂物 ignore | — | ready |
| h2 | `task_5b5c35c90c54` | 依赖定版：gorm 声明与构建一致 + 两种库形态冒烟 | — | ready |
| h3 | `task_03aeebddf8c8` | CI 与测试入口：`make test-integration/lint/vet` + harness 机检 | h1,h2 | pending |
| h4 | `task_0fd4798db4ea` | 推送越权：**两处**入口不再信任客户端 user_id | h3 | pending |
| h5 | `task_af7088ce6598` | 静默吞错：5 处留痕 + 指标 | h3 | pending |
| h6 | `task_fca2e51dbf2a` | 金额口径审计：三套口径 + 跨界换算点（只读） | **无**（放宽） | ready |
| h6b | `task_ff6d29793eaf` | 落地：`transactions` 收敛到整数分（JSON 契约不变） | h6 | pending |
| h7 | `task_f10481ca877d` | 环境名静默回退 + `/metrics` 收口 | h3 | pending |
| h8 | `task_92adb13b2632` | membership 核销器：回调主动发权益（F2） | h3 | pending |

## 人已批的决策（2026-11-06，已写进对应 Brief 与 Task spec）

1. **派发方式**：只建 Task DAG，先不 worker-start。
2. **h2 gorm**：接受 `v1.31.2`，把 `app/go.mod:20` 从 `v1.25.12` 回写（HEAD 本就在编 1.31.2，属「把既成事实写进声明」）。
3. **h6**：方案 **+** 落地 `transactions` 一条链路 → 拆 h6（只读）+ h6b（落地）；边界：对外 JSON 单位不变、Supabase/ES 路径本轮不动、旧列不删。
4. **杂物**：`app/worker`（39MB 二进制）只 ignore 不删除；`/teach` 产物保持在库；`app/internal/rag/` 零测试本轮不拆 Slice。

**唯一未决门**（已写进 h3 的 Task spec，要求 worker 动手前先 `ask`）：CI 落点 GitHub Actions vs 本地 `make` + githooks；integration 设为阻塞门禁还是先 allow-fail。

## Notes

- **coordinator 本轮只产出规划文档 + Task DAG，零 `.go` 改动**（`git diff --name-only | grep '\.go$'` 为空）
- **前提变更（2026-11-05 提交 `3f0963e`）**：原盘点「`rag`/`authz`/`txdb`/`schemaboot`/`payment.go`/`migrations`/`plans`/`docs` 未跟踪、HEAD 编不过」已解除；`go build all` 在 HEAD 实测通过。H1 范围收窄为「import 覆盖机检 + `app/worker`（39MB 二进制）杂物 ignore」；风险 R1 标注已解除
- **四处取证更正**已写入 Brief，防止 worker 按错前提干活：
  0. H4 —— 越权是**两处**（`jpush_controller.go:91-93` + `realtime_controller.go:109-112`），不是一处；H6 —— 金额口径是**三套**（整数分 / decimal-as-string / float64），mall 不是分口径，且 `transactions.amount` 在 DB 侧是**裸 `numeric`（无精度）**、全链路零 `Round(_,2)`
  1. H7 —— `configs/config.prod.yaml` **存在**；真问题是 `APP_ENV=production` 找不到同名文件时**静默回退 `config.yaml`（debug + sslmode disable）**（`common/config/config.go:844-852`）
  2. H8 —— `main.go:246` 已把 `paymentSvc` 注入 `membership.NewService`，注册可在包内完成，**零 main.go 改动** → h8 可与 h4/h5/h7 并行
- **新发现且未排 Slice**：`app/internal/rag/` 5 个文件零测试（Program §背景 第 10 条），已显式登记不静默丢弃
- 待处置杂物：`app/worker` 39MB Mach-O 可执行文件（未 ignore，**未删除**，等人决定）
- 并行建议：h1、h2 可同时起步；h3 完成后 h4/h5/h6/h7/h8 五条可同批派发（文件锁互不重叠）
- 遗留人证（沿上一轮）：lan-up / 飞书 / ES / queue / Agent curl 均为 Partial Deferred；Flutter 客服页未做
- scratch 库待清理（Docker 恢复可达后，且只删本轮自建的）：`p01_*` / `p02_it` / `p02_mig` / `p02_smoke` / `p03_it`

## 历史指针

| 时间 | Active | 备注 |
|------|--------|------|
| 2026-11-04 前 | `cs-agent-infra` C1–C4 | C1 `bd116e1` · C2 `089dfa1` · C3 `748e517` · C4；lan-up/飞书/ES/queue/Agent curl 人证 Partial Deferred |
