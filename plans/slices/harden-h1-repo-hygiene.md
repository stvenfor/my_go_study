# Slice — harden-h1-repo-hygiene

> **Brief 修订（2026-11-05）**：本 Slice 的原前提「两个已装配包不在 git 里、HEAD 编不过」**已被提交 `3f0963e` 解除**。
> 复测证据：`git ls-files app/internal/rag core/authz app/internal/txdb app/internal/schemaboot model/entity/payment.go` 全部命中；未跟踪项由 73 → **6**；`go build all` 在 HEAD **通过**（6.6s）。
> 因此范围收窄为「机械化复核 + 剩余杂物」，不再是「抢救式入库」。Program §背景 第 1 条与风险 R1 同步标注为已解除。

## Slice Brief
- SOURCE_MODULE: 仓库根（git 可见性；不改代码）
- TARGET_MODULE: `.gitignore` + 一份「import 覆盖」机检证明
- Source entry: 剩余未跟踪 6 项 = `app/worker`（**39MB Mach-O 可执行文件**，`git check-ignore -v app/worker` 无命中 → 未被忽略）+ 本 Program 的 5 份 slice brief（h4–h8）
- Target entry: ①「每个被 import 的工作区包都有 tracked 文件」是**脚本证明**的，不是人眼看的 ②`app/worker` 这类编译产物被 ignore ③干净 worktree 从 HEAD 检出 `go build all` 通过
- 本轮 ONLY: ①写一个可重复跑的机检（`go list -m all` + `go list -deps` 本地包 vs `git ls-files`，输出缺失列表，期望 0）②`.gitignore` 补编译产物模式 ③干净 worktree 复证 ④杂物归属结论落 acceptance-record
- 不做:
  - **不删除 `app/worker`**（删除属破坏性动作，且在 `app/` 模块内易误伤）→ 只 ignore，并把它写进残留清单请人处置
  - 不改任何 `.go`；不碰 `go.mod`；不动 `.env*`
  - 不再重做「73 项三分清单」（已完成于 `3f0963e`），只复核
- 验收:
  - [ ] 机检脚本跑通并输出「缺失 tracked 的本地 import 包 = 0」；脚本本身入库（`scripts/`），可被 CI/harness 复用
  - [ ] `git check-ignore -v app/worker` 命中；`git status --short` 未跟踪项不再包含任何二进制产物
  - [ ] `git worktree add /tmp/h1-verify $(git rev-parse HEAD)` → 在该目录 `go build all` **通过**（与原 R1 预期相反，即风险已解除的正面证据）；完成后 `git worktree remove /tmp/h1-verify`
  - [ ] 大文件复核：`git ls-files` 中无 >1MB 文件（本轮实测 0 命中）→ 作为规则写进机检，防以后误提交二进制
  - [ ] `/teach` 产物归属结论：`MISSION.md`、`NOTES.md`、`learning-records/`、`reference/`、`assets/`、`data/cs-kb/` **已随 `66ece97` 入库**（实测 tracked 11 项）→ 决策门 2 收敛为「保持入库」（默认）vs「移出仓库」，由人拍板并记录
- 文件白名单:
  - `.gitignore`
  - `scripts/`（新增机检脚本）
  - `docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md`
  - `plans/slices/harden-h1-repo-hygiene.md`（仅 Context Card）
- 文件黑名单:
  - `app/**`、`core/**`、`model/**`、`common/**` 下任何 `.go`
  - `**/go.mod`、`**/go.sum`
  - `app/worker`（**只 ignore，不删、不提交**）
  - `.env`、`.env.local`、`.env.lan`
- 验证命令:
  - `./scripts/check-tracked-imports.sh`（本 Slice 新建）
  - `git check-ignore -v app/worker`
  - `git worktree add /tmp/h1-verify $(git rev-parse HEAD) && (cd /tmp/h1-verify && go build all) && git worktree remove /tmp/h1-verify`
  - `git status --short`（未跟踪项应为 0，或仅人明确保留的）
- 证据: `docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md`

## 决策门（2026-11-06 已批，不再阻塞）
1. `app/worker` 39MB 二进制 → **只 ignore，不删除**（人已批）。产物仍留在工作区，Brief 里必须写清「未删除」这一事实，不许含糊成「已处理」。
2. `/teach` 产物（`MISSION.md`/`NOTES.md`/`learning-records/`/`reference/`/`assets/`/`data/`）→ **保持在库**（人已批）。
3. 新发现的 `app/internal/rag/` 5 文件零测试 → **本 Program 不拆 Slice**（人已批），只作为 §背景 第 10 条留痕。

## Context Card — harden-h1-repo-hygiene
- 已完成:
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
