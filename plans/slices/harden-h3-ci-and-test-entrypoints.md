# Slice — harden-h3-ci-and-test-entrypoints

## Slice Brief
- SOURCE_MODULE: 仓库根（`Makefile`、`scripts/`、`.github/`、`.harness/`）
- TARGET_MODULE: 同源的 CI 门禁 + 本地入口
- Source entry: `Makefile:46-56`（`test`/`test-all` 只跑 `go test ./...`，**不含 `-tags=integration`**）；`grep -rn 'tags=integration' Makefile scripts/` → 0 命中；`.github/` 目录**不存在**；`grep -n 'agent-pre\|agent-post\|check-brief' Makefile` → 0 命中（`.harness/rules/slice-contract.md` 承诺的机检从未落地）
- Target entry: `make test-integration` / `make lint` / `make vet` / `make agent-pre` 四条可跑；CI 在 PR 上真跑并拦住过故意制造的失败
- 本轮 ONLY: 落点 GitHub Actions（新建 `.github/workflows/ci.yml`）+ 四条本地门禁入口 `make test-integration` / `make lint` / `make vet` / `make agent-pre`；阻塞项 = `go build all` / `go vet` / lint / 单元测试 / `scripts/check-tracked-imports.sh`，集成测试 `continue-on-error: true` allow-fail 起步。
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

## 决策门（2026-11-06 已批）

- **落点 = GitHub Actions**（新建 `.github/workflows/ci.yml`），不做本地 githooks 方案。
- **门禁分层**：`go build all` / `go vet` / lint / 单元测试 / `scripts/check-tracked-imports.sh` = **阻塞项**；**集成测试（`-tags=integration`）先 `continue-on-error: true`（allow-fail）**，等它连续绿若干次再单独提一次改动转成阻塞 —— 转阻塞不在本 Slice。
- **本仓无外网**（coordinator 实测 `curl https://github.com` 之外的 `proxy.golang.org` 返回 000）：worker 只能**写 workflow 文件 + 本地等价复现**（同一组命令在本机跑通/跑挂各一次），真实 Actions 绿灯需人 push 后回看，acceptance-record 里把这条写清为 Partial 的原因。

### 新决策门（2026-11-06 18:0x，coordinator 实测后追加）：toolchain 必须先钉死，否则 `go build all` 阻塞项永远红

- 实测：本机默认 **go1.27.3** 下，`go build all`（主仓、有 `go.work`）**exit 1**：
  `grpc v1.71.0/internal/transport/handler_server.go:271 undefined: http2.TrailerPrefix`
  —— workspace MVS 选到 `x/net v0.54.0`，该符号在 go1.27 下被构建标签编译掉。`GOTOOLCHAIN=local` + `/usr/local/go`(1.26.4) 时 **exit 0**。
- 因此本 Slice 的 CI 必须显式钉 toolchain（建议 `setup-go` 的 `go-version: '1.26.x'`，并在 acceptance-record 写明「不是用作者机器的 1.27.3」）；**禁止**靠 `GOTOOLCHAIN=auto` 让 CI 漂到 1.27。
- 另一种修法（升 grpc / 钉 `x/net` / `-tags http2legacy`）会碰 `**/go.mod` = 本 Slice 黑名单，故**不在本 Slice**；若人要选那条，另开依赖 Slice（h2 锁区）。
- 验收补充：`go build all` 在本 Slice 的 CI 下必须绿，且 acceptance-record 必须贴 `go version` 与 `go env GOTOOLCHAIN`；只贴 exit 0 不贴版本视为证据不足。

## 决策门（历史）（人批后才动手）
- CI 跑在哪：GitHub Actions（需确认 remote 与 Actions 额度）/ 只做本地 `make` 门禁 + githooks（`scripts/install-githooks.sh` 已存在，可复用）。
- 是否需要把 integration 用例设为**阻塞**门禁（真库依赖会让 CI 变慢变脆）还是 allow-fail 起步。

## Context Card — harden-h3-ci-and-test-entrypoints
- 已完成:
  - Makefile 四入口：`test-integration` / `lint` / `vet` / `agent-pre`（沿用 `for m in $(MODULES)` 风格）
  - `.github/workflows/ci.yml`：build 阻塞 job（钉 `go-version: '1.26.x'` + `GOTOOLCHAIN: local`）+ integration allow-fail job（PG16 service、独立库 `h3_it`）
  - `scripts/gofmt-check.sh` + `gofmt-baseline.txt`（不新增欠账 ratchet）、`scripts/test-integration.sh`（一次性容器 / 外部库双模式）、`scripts/check-brief.sh`（六节 + 缩进 + `--self-test`）
  - `.harness/rules/slice-contract.md` 机检落地说明；acceptance-record 已写
  - 本机等价复现全绿（1.26.4）：`go build all` / vet / lint / unit / agent-pre / integration 全 exit 0；lint/unit/integration 各注入一次失败并全部还原
- 未做/Deferred:
  - 真实 GitHub Actions 绿灯（无外网、推不推由人；需人 push 后回看）→ record 记 Partial
  - integration 转阻塞（决策门：allow-fail 起步，另开 Slice）
  - 格式化 10 个既有未 gofmt 文件并清空 baseline（业务 .go 黑名单，另开 Slice）
- 关键文件:
  - `Makefile:60-87`（四 target）、`.github/workflows/ci.yml`、`scripts/{gofmt-check,gofmt-baseline,test-integration,check-brief}.*`
  - `docs/acceptance-records/2026-11-05-harden-h3-ci-and-test-entrypoints.md`
- harness: post ok?
  - `make agent-pre`（check-brief）对全部 27 个 slice 零误报；`--self-test` PASS；未跑 `agent-post`（本仓未落地，见 unit-test-ci skill）
- 下一 Slice 建议:
  - h1/h2 合并后按 Task DAG 顺序回看 Actions；h1 的 `check-tracked-imports.sh` 入库后该阻塞项转绿
  - integration 连续绿若干次后单独提「转阻塞」；格式化欠账文件收紧 gofmt 门禁
- 已知坑:
  - 默认 `go`=1.27.3 会触发 grpc/x-net `http2.TrailerPrefix` 漂移；本地必须 `PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local`
  - workspace `go vet all` 含依赖预存告警，故 `make vet` 用逐模块等价
  - 一次性 PG 容器就绪有竞态（已硬化探活）；bash 3.2 `"${s}"` 定界；zsh 无词分割 / `PIPESTATUS` 差异
