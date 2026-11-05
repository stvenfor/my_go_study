# Slice — harden-h1-repo-hygiene

## Slice Brief
- SOURCE_MODULE: 无（本 Slice 不动代码，只动 git 可见性）
- TARGET_MODULE: 整个工作区（`app` / `core` / `model` / `common` / `migrations` / `plans` / `docs`）
- Source entry: `git status --short` 未跟踪 73 项；已装配进主程序却不在库的包 `app/internal/rag/`、`core/authz/`（引用点 `app/cmd/api/main.go:54,61,655-714`、`app/internal/access/service.go:19`）
- Target entry: `git ls-files` 覆盖所有被 import 的包；`git worktree` 干净检出 `go build all` 通过
- 本轮 ONLY: ①盘点未跟踪文件并三分（必入库 / 必 ignore / 待人裁决）②改 `.gitignore` 收口杂物 ③产出**待批**的 `git add` 命令清单（**不执行 add/commit**）④用干净 worktree 证明「HEAD + 待入库文件」能编译
- 不做: 不改任何业务逻辑；不碰 `go.mod`；不删任何未跟踪文件（删除属破坏性动作，交人）；不动 `.env*`
- 验收:
  - [ ] `./scripts/audit-untracked.sh`（或等价一次性命令）输出三分清单，落盘 acceptance-record 附表
  - [ ] `.gitignore` 新增条目逐条有理由（构建产物 / IDE / 密钥 / 工具索引），并证明 `git check-ignore -v` 命中
  - [ ] 干净 worktree：`git worktree add /tmp/h1 $(git rev-parse HEAD)` → 只叠加「待入库文件」→ `go build all` 通过（**当前 HEAD 单独检出必须失败**，否则说明问题不成立）
  - [ ] 反证**只用 worktree**（禁用 `git stash`，避免动到 159 个未提交改动）：`git worktree add /tmp/h1-head $(git rev-parse HEAD)` → 在该目录 `go build all` **必须失败**，错误指向 `rag`/`authz` 缺失 → 证明 R1 风险真实；完成后 `git worktree remove /tmp/h1-head`
  - [ ] 待批 `git add` 命令清单写进 acceptance-record，由人执行
- 文件白名单:
  - `.gitignore`
  - `docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md`
  - `plans/slices/harden-h1-repo-hygiene.md`（仅 Context Card）
- 文件黑名单:
  - `app/`、`core/`、`model/`、`common/` 下任何 `.go`
  - `**/go.mod`、`**/go.sum`
  - `.env`、`.env.local`、`.env.lan`
- 验证命令:
  - `git status --short | wc -l`
  - `git ls-files app/internal/rag core/authz | wc -l`（应为 0 → 完成待批 add 后 >0）
  - `go build all`
  - `git check-ignore -v build node_modules .codegraph`
- 证据: `docs/acceptance-records/2026-11-05-harden-h1-repo-hygiene.md`

## 决策门（人批后才算完成）
1. 待入库范围（含 `plans/`、`docs/`、`migrations/*.sql` 是否整批入库）。
2. `/teach` 产物归属：`NOTES.md`、`learning-records/`、`reference/`、`assets/`、`data/` → 入库 / ignore / 移出仓库（Program 决策门 2）。
3. 由人执行 `git add` + commit；本 Slice **不自动 stage**。

## Context Card — harden-h1-repo-hygiene
- 已完成:
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
