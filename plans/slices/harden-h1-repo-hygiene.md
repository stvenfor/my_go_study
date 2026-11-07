# Slice — harden-h1-repo-hygiene

> **Brief 修订（2026-11-05）**：本 Slice 的原前提「两个已装配包不在 git 里、HEAD 编不过」**已被提交 `3f0963e` 解除**。
> 复测证据：`git ls-files app/internal/rag core/authz app/internal/txdb app/internal/schemaboot model/entity/payment.go` 全部命中；未跟踪项由 73 → **6**；`go build all` 在 HEAD **通过**（6.6s）。
> 因此范围收窄为「机械化复核 + 剩余杂物」，不再是「抢救式入库」。Program §背景 第 1 条与风险 R1 同步标注为已解除。

## Slice Brief
- SOURCE_MODULE: 仓库根（git 可见性；不改代码）
- TARGET_MODULE: `.gitignore` + 一份「import 覆盖」机检证明
- Source entry: 剩余未跟踪 6 项 = `app/worker`（**39MB Mach-O 可执行文件**，`git check-ignore -v app/worker` 无命中 → 未被忽略）+ 本 Program 的 5 份 slice brief（h4–h8）
- Target entry: ①「每个被 import 的工作区包都有 tracked 文件」是**脚本证明**的，不是人眼看的 ②`app/worker` 这类编译产物被 ignore ③~~干净 worktree 从 HEAD 检出 `go build all` 通过~~ → **改为「纯跟踪文件构树能编过」为验收**，且当前 HEAD **编不过**（`thunder/logs` 被 `logs/` 规则误伤），这是本 Slice 第一个真修复项
- 本轮 ONLY: ①写一个可重复跑的机检（`go list -m all` + `go list -deps` 本地包 vs `git ls-files`，输出缺失列表，期望 0）②`.gitignore` 补编译产物模式 ③干净 worktree 复证 ④杂物归属结论落 acceptance-record
- 不做:
  - **不删除 `app/worker`**（删除属破坏性动作，且在 `app/` 模块内易误伤）→ 只 ignore，并把它写进残留清单请人处置
  - 不改任何 `.go`；不碰 `go.mod`；不动 `.env*`
  - 不再重做「73 项三分清单」（已完成于 `3f0963e`），只复核
- 验收:
  - [ ] 机检脚本跑通并输出「缺失 tracked 的本地 import 包 = 0」；脚本本身入库（`scripts/`），可被 CI/harness 复用
  - [ ] `git check-ignore -v app/worker` 命中；`git status --short` 未跟踪项不再包含任何二进制产物
  - [ ] ~~`git worktree add /tmp/h1-verify $(git rev-parse HEAD)` → 在该目录 `go build all` 通过（风险已解除的正面证据）~~ → **coordinator 于 2026-11-06 作废本断言并改为反向取证**。亲跑结果：`git archive HEAD | tar -x -C <tmp>` + 手写 `go.work` → `go build all` **exit 1**，报 `no required module provides package github.com/mszlu521/thunder/logs`。两条根因：`.gitignore:11` 裸 `logs/` 把源码包 `thunder/logs/` 一起忽略了；`.gitignore:33-34` 忽略 `go.work`/`go.work.sum`，干净树无 workspace 文件。本条改为：
      ① 先收窄 `logs/` 规则，使 `git check-ignore -v thunder/logs/logs.go` **无输出**（即该源码包可被跟踪）；
      ② 再重跑同一严格复现，把**真实退出码与报错原文**写进 acceptance-record（修好前预期仍是 exit 1，这正是要暴露的缺陷，不许为了让绿灯而伪造）；
      ③ **决策门已批（2026-11-06 人批：`go.work` 入库）** → 你可以改 `.gitignore` 放行 `go.work`（`:33` 那行删掉或改成带例外的规则），并按同一口径决定 `go.work.sum`（`:34`）——**倾向一并入库**，因为无外网环境里 workspace 校验和缺失会让 CI 首次构建拉不到已缓存的模块校验记录；把你的判断与理由写进 acceptance-record。**仍然禁止你自己 `git add go.work` / `git add go.work.sum`**：跟踪动作由人执行，你只负责规则与证据。
      ④ 放行后的验收：`git archive HEAD | tar -x -C <tmp>` 后**不要手写 go.work**，直接 `go build all` → 期望 **exit 0**。注意在你这条分支上，只要人还没 `git add`，此步必然仍失败 —— 那就把它写成「规则已改、待 add 后即绿」的 Partial，并附 `git check-ignore -v go.work` **无输出** 作为「已放行」的直接证据。
      ④ 复现一律用 `git archive HEAD | tar -x`（比 `git worktree add` 更严格：后者同样拿不到 ignored 文件，但会额外创建分支，你已因此差点跑 `git branch -D`）
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
  - `T=$(mktemp -d) && git archive HEAD | tar -x -C "$T" && (cd "$T" && printf 'go 1.25.0\n\nuse (\n\t./common\n\t./core\n\t./model\n\t./app\n\t./thunder\n)\n' > go.work && go build all); echo exit=$?   # 严格复现：只用跟踪文件；临时目录用 mktemp -d 且**不要 rm -rf**（本机 rm -rf 每次弹高风险审批，会把你锁死）`
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
