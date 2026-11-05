# Slice — harden-h3-ci-and-test-entrypoints

## Slice Brief
- SOURCE_MODULE: 仓库根（`Makefile`、`scripts/`、`.github/`、`.harness/`）
- TARGET_MODULE: 同源的 CI 门禁 + 本地入口
- Source entry: `Makefile:46-56`（`test`/`test-all` 只跑 `go test ./...`，**不含 `-tags=integration`**）；`grep -rn 'tags=integration' Makefile scripts/` → 0 命中；`.github/` 目录**不存在**；`grep -n 'agent-pre\|agent-post\|check-brief' Makefile` → 0 命中（`.harness/rules/slice-contract.md` 承诺的机检从未落地）
- Target entry: `make test-integration` / `make lint` / `make vet` / `make agent-pre` 四条可跑；CI 在 PR 上真跑并拦住过故意制造的失败
- 验收:
  - [ ] `make test-integration`：用 `docker run` 起一次性 PG16（或复用 `my_go_study_postgres` 上的独立 scratch 库，**绝不连业务库 `my_go_study`**），跑现有 5 条 integration 用例，全绿
  - [ ] `make vet`：`go vet all` 或逐模块等价，当前必须 0 错（有错则先记录、不顺手改业务代码）
  - [ ] `make lint`：选型只用现成依赖（`gofmt -l` + `go vet` 起步；要加 golangci-lint 必须 pin 版本，参照 `Makefile:66` 的 `MIGRATE_VERSION ?= v4.15.2` 写法）
  - [ ] `make agent-pre`：实现 `.harness/rules/slice-contract.md` 的 `check-brief` 机检 —— 校验 `plans/slices/<id>.md` 含「本轮 ONLY / 不做 / 验收 / 文件白名单 / 验证命令 / 证据」六节且子列表有缩进；对**已有全部 slice 文件**跑一遍，不误报
  - [ ] CI：新增 workflow（`go build all` + `go test ./...` + `-tags=integration` job 带 PG service + `make agent-pre`）。**本仓两个 remote 之前没有 CI，推不推由人决定**（见决策门）
  - [ ] **故意制造一次失败并观察到被拦**：lint（塞一个未 gofmt 文件）、单测（改一个断言）、integration（塞一个 `-tags=integration` 失败用例）各一次，记录输出后全部还原
  - [ ] `git diff` 证明本 Slice 只动 `Makefile` / `scripts/` / `.github/` / `.harness/`
- 不做: 不改任何业务代码与测试断言语义；不引入新的 Go 依赖进 `app`/`core`（h2 刚定版口径）；不建发布流水线
- 文件白名单:
  - `Makefile`
  - `scripts/`
  - `.github/workflows/`
  - `.harness/**`（仅机检脚本与新 rule 说明）
  - `docs/acceptance-records/2026-11-05-harden-h3-ci-and-test-entrypoints.md`
  - `plans/slices/harden-h3-ci-and-test-entrypoints.md`（仅 Context Card）
- 文件黑名单:
  - `app/**/*.go`、`core/**/*.go`、`model/**`、`common/**`
  - `**/go.mod`、`**/go.sum`
- 验证命令:
  - `make vet && make lint && make test`
  - `make test-integration`
  - `make agent-pre`
  - 本地等效 CI：`go build all && go -C app test ./... -count=1`
- 证据: `docs/acceptance-records/2026-11-05-harden-h3-ci-and-test-entrypoints.md`

## 决策门（人批后才动手）
- CI 跑在哪：GitHub Actions（需确认 remote 与 Actions 额度）/ 只做本地 `make` 门禁 + githooks（`scripts/install-githooks.sh` 已存在，可复用）。
- 是否需要把 integration 用例设为**阻塞**门禁（真库依赖会让 CI 变慢变脆）还是 allow-fail 起步。

## Context Card — harden-h3-ci-and-test-entrypoints
- 已完成:
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
