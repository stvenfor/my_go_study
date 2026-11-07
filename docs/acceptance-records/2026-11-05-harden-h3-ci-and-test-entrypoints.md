# Acceptance Record — harden-h3-ci-and-test-entrypoints

- 实施/验收日期：2026-11-07（本机执行日；文件名沿用 Brief 指定的 2026-11-05）
- Slice：`plans/slices/harden-h3-ci-and-test-entrypoints.md`
- TaskId：`task_03aeebddf8c8`
- 角色：h3 worker（executor；coordinator 只验收）
- 自证：`pwd` = `/Users/qpy/orca/workspaces/my_go_study/harden-h3`；`git rev-parse --short HEAD` = `e288553` —— 均满足，未 ask。
- 主交付物：`Makefile`（4 入口）、`.github/workflows/ci.yml`、4 个 `scripts/*`、`.harness/rules/slice-contract.md`（1 行）、本 record、slice Context Card。

## status: **Degraded**

note（拆开口径，避免「本机全绿 = 真 CI 绿」）：
1. 4 条 `make` 入口 + CI workflow + 3 个新脚本 + `check-brief` 机检全部落地；本机用**钉死的 1.26.4** 跑通全部阻塞门禁、跑挂三次故意注入的失败、integration 真库全绿（§3–§6）。
2. **唯一未闭项 = 真实 GitHub Actions 在 PR 上跑绿**。决策门（2026-11-06 已批）：本仓 worker 无外网，只能「写 workflow 文件 + 本地等价复现（同组命令本机跑通/跑挂各一次）」，推不推由人决定，绿灯需人 push 后回看。故按 **Partial** 记，不勾满、不自称 Accept。

- Mode: **Verified（本机等价复现，全绿）** / Partial（真实 Actions 绿灯待人 push 回看）

## 0. 前提核对（都是未入库的兄弟产物，未误判）

| 项 | 处理 |
|---|---|
| `go.work` / `go.work.sum` / `thunder/logs/logs.go` / `bin/migrate` | coordinator 复制进来的构建前置（未跟踪），**非本 worker 改动，未 `git add`** |
| `scripts/check-tracked-imports.sh` | h1 的未入库产物，**预热用，随 h1 入库**；本轮**不修改、不提交**，仅用于本地 CI 预演（§3） |
| h2 的 gorm v1.31.2 | 未入库；本 worktree `app/go.mod` 仍 `v1.25.12`。`go.mod`/`go.sum` 在黑名单**未改**；验收按当前 HEAD 声明跑。**h2 合并后声明会变、构建行为不变**（§10） |
| Brief 顶部「新决策门：toolchain 必须先钉死」 | coordinator 刚追加的未入库改动，**已读并据此钉 1.26.x**（§1）；该 diff 非本 worker 产出 |

- 边界复核：未改任何业务 `.go`（`git diff --name-only -- '*.go'` 为空）、未改 `go.mod`/`go.sum`、未 commit/push、未 `rm -rf`（仅 `rm <单文件>` / `docker rm -f <自建容器>`；`Makefile:211` 的 `rm -rf` 是既有 `clean` target，未运行）。
- DB：只用一次性容器 `h3-it-pg-*`（`POSTGRES_DB=h3_it` 容器内自建独立库）；本轮未起任何 HTTP 服务，`:8080`/`:8081` 均未动；业务库 `my_go_study` 全程未连。

## 1. 新决策门：toolchain 必须先钉死

- 本机默认 `go` = **1.27.3**（`/usr/local/bin/go`），`GOTOOLCHAIN=auto`。`go build all` **exit 1**：
  ```
  # google.golang.org/grpc/internal/transport
  .../grpc@v1.71.0/internal/transport/handler_server.go:271:18: undefined: http2.TrailerPrefix
  ```
  （workspace MVS 选到 `x/net v0.54.0`；环境事实，非本 Slice bug）
- CI 钉死：`setup-go` `go-version: '1.26.x'`（两个 job 各一处）+ workflow 级 `env: GOTOOLCHAIN: local`，**禁止 auto 漂到 1.27**；两个 job 均打印 `go version` / `go env GOTOOLCHAIN`。
- 本机等价复现（`/usr/local/go/bin/go` + `GOTOOLCHAIN=local`）：
  ```
  go version go1.26.4 darwin/amd64
  go env GOTOOLCHAIN = local
  GO1264_BUILD_ALL_EXIT=0
  ```
  —— **不是用作者机器的 1.27.3**；按验收补充，版本与 GOTOOLCHAIN 均已贴，不只贴 exit 0。

## 2. 交付物

| 路径 | 作用 |
|---|---|
| `Makefile` | 新增 `.PHONY` 项与 `test-integration` / `lint` / `vet` / `agent-pre` 四 target（`Makefile:60-87`，紧接原 `test` 之后，沿用现有 `for m in $(MODULES)` 风格） |
| `.github/workflows/ci.yml` | `build`（阻塞：`go build all` / `make vet` / `make lint` / `make test` / `check-tracked-imports.sh` / `make agent-pre`）+ `integration`（PG16 service、`continue-on-error: true`） |
| `scripts/gofmt-check.sh` + `scripts/gofmt-baseline.txt` | `gofmt -l` 门禁（不新增欠账 ratchet） |
| `scripts/test-integration.sh` | 一次性 PG16 容器 / CI 外部库双模式，跑 `-tags=integration` |
| `scripts/check-brief.sh` | Slice 合同机检（六节 + 子列表缩进）+ `--self-test` |
| `.harness/rules/slice-contract.md` | 「机检（若已落地）」→ 落地说明（1 行） |

## 3. 阻塞门禁：本机等价复现（全部 1.26.4）

```
make vet          EXIT=0
make lint         EXIT=0  (OK: gofmt 门禁通过（既有 10 个欠账文件已登记，未新增；扫描 395 个 .go） + go vet)
make test (unit)  EXIT=0  (38 个 ok)
make agent-pre    EXIT=0  (OK: check-brief 通过（27 个 slice 六节齐全、子列表缩进）)
go build all      EXIT=0
```
- `check-tracked-imports.sh`（h1 预热）：主检 **exit 1** —— `MISSING thunder/logs（.gitignore:11:logs/ 仍在忽略）`；`--self-test` **exit 0** `[self-test] PASS`。原因与 h1 依赖见 §9/§10。
- 门禁分层（决策门）：`go build all` / `go vet` / lint / unit / `check-tracked-imports.sh` / `agent-pre` = **阻塞**；`-tags=integration` job = `continue-on-error: true`（allow-fail 起步，转阻塞另开 Slice）。
- CI YAML 经 Ruby `YAML.load_file` 解析通过；`go-version: '1.26.x'` ×2、`GOTOOLCHAIN: local` ×1、`continue-on-error: true` ×1 均在位。

## 4. 故意制造失败 —— 各一次、贴原始输出、全部还原

### lint（塞一个未 gofmt 文件）
注入 `scripts/z_gofmt_probe.go`（untracked、非 ignore）：
```
MAKE_LINT_EXIT=2
FAIL: 以下 .go 未 gofmt，且不在 baseline 欠账清单:
  scripts/z_gofmt_probe.go
修法: gofmt -w <文件>；确属历史欠账再登记进 .../scripts/gofmt-baseline.txt
make: *** [lint] Error 1
```
→ `rm scripts/z_gofmt_probe.go`。

### unit（改一个断言）
临时把 `common/jwt/jwt_uuid_test.go:22` 的 `a@b.com` 改成 `INJECTED-WRONG@example.com`：
```
MAKE_TEST_EXIT=2
--- FAIL: TestGenerateAndParseUUID (0.00s)
    jwt_uuid_test.go:23: email="a@b.com"
FAIL
FAIL	common/jwt	1.223s
make: *** [test] Error 1
```
→ 从 `/tmp` 备份精确还原，`git diff --exit-code -- common/jwt/jwt_uuid_test.go` 为空。

### integration（塞一个 -tags=integration 失败用例）
注入 `app/internal/payment/z_h3_injected_integration_test.go`（`//go:build integration` + `t.Fatal`）：
```
MAKE_TEST_INTEGRATION_EXIT=2
--- FAIL: TestH3InjectedIntegrationFailure (0.00s)
    z_h3_injected_integration_test.go:9: H3 injected integration failure (proof only, removed after)
FAIL
FAIL	app/internal/payment	1.672s
make: *** [test-integration] Error 1
```
→ `rm` 该文件；无残留容器。

- 还原总证：`git diff --name-only -- '*.go'` = 空；`git ls-files --others --exclude-standard` 无 probe 文件；`docker ps -a` 无 `h3-it-*`。
- 过程注记：integration 首次注入时 readiness 偶发 `PG 未就绪`（容器启动竞态，失败点没落到注入用例），已把 `test-integration.sh` 就绪循环硬化（容器存活检查 + 60 次重试 + 失败打 `docker logs`），重跑后失败点正确落到注入用例；硬化后干净跑仍 exit 0（§5）。

## 5. `make test-integration`（一次性 PG16 + 独立 scratch 库）

- 脚本两种模式：本机默认 `docker run` 一次性 `postgres:16-alpine`（`POSTGRES_DB=h3_it` 容器内自建独立库、随机宿主端口、trap 删除）；CI 用 `H3_IT_EXTERNAL_DB=1` 指向 `services.postgres`（同样 `POSTGRES_DB: h3_it`，**不连 my_go_study**）。
- 两个库名 env（`P02_PG_DBNAME` / `P03_PG_DBNAME`）都指 `h3_it`，配合 `-p 1` 串行避免跨包 DDL 竞态。
- 命令：`go -C app test -tags=integration -p 1 ./... -count=1`。
- 最终干净跑：`MAKE_TEST_INTEGRATION_EXIT=0`，`30` 个 `ok` 包，无残留容器。
- **`-v` 复证用例真的跑了（不是 `t.Skipf` 假绿）**：`RUN=60 PASS=60 SKIP=0 FAIL=0`，12 条 `//go:build integration` 用例全部真库 PASS（如 `TestPostgresConcurrentNotifySettlesOnce 0.08s`、`TestChannelNotifySettlesMallOrder 0.25s`、`TestMembershipBalanceTxRollsBackDebit 0.10s`）。
- Brief 写「5 条 integration 用例」，**实测 12 条**（分布在 `payment`/`mall`/`membership` 三个包的 4 个文件）；按当前 HEAD 全跑全绿，未因数字差异缩范围。

## 6. `make agent-pre` / check-brief 机检（六节 + 全 slice 不误报）

- `scripts/check-brief.sh`：
  - 六节（`本轮 ONLY` / `不做` / `验收` / `文件白名单` / `验证命令` / `证据`）用**字面存在**判定 —— 人批 Brief 有把标签写在引号枚举里的变体（harden-h3 自身即如此），强解析 `^- 本轮 ONLY:` 会误杀已批 Brief，故取「拦整节漏写」的宽松口径以保「不误报」。
  - 「子列表缩进」只查 `## Slice Brief` 区：该区顶格 `- ` 行必须是字段头（`- 名字:`），否则即未缩进子项；`## 决策门` / `## Context Card` 区的顶格列表项不受此约束，避免误报。
  - `--self-test` 造 3 个临时样例自证「合法放行 / 缺节拦下 / 未缩进拦下」全 PASS。
- `make agent-pre` 对**全部 27 个** `plans/slices/*.md`（含 `_template.md`）跑一遍：`OK: check-brief 通过（27 个 slice 六节齐全、子列表缩进）`，**零误报**。
- `.harness/rules/slice-contract.md` 把「机检（若已落地）」改为落地说明（1 行）。

## 7. gofmt -l 为什么用「不新增欠账」ratchet

- HEAD 上有 **10 个 tracked `.go`** 本就未 gofmt（import 分组顺序），业务 `.go` 属本 Slice 黑名单不能改。裸 `gofmt -l $(git ls-files '*.go')` 会让 CI 一落地就恒红，既无法「在 PR 上真跑」，也无法观察故意注入的失败。
- 故 `scripts/gofmt-baseline.txt` 登记这 10 个既有欠账（1.26.4 与 1.27.3 gofmt 输出逐行一致），`scripts/gofmt-check.sh` 扫 `tracked + 未忽略 untracked` 的 `.go`，扣掉 baseline 后只拦**新增**欠账；新塞的 `scripts/z_gofmt_probe.go` 即被拦（§4）。
- `make lint` = `gofmt-check.sh` + `make vet`（Brief 指定「gofmt -l + go vet 起步」，未引入 golangci-lint）。
- 后续另开格式化 Slice 对这 10 个文件 `gofmt -w` 并从清单移除即可收紧。

## 8. `make vet` 为什么是逐模块等价

- `go vet all`（workspace `all` 含依赖）会把 `protobuf` 的 unkeyed fields、`gorm` 的 context leak 等**依赖模块预存告警**一起报出来（实测 1.26.4 下大量输出）。Brief 允许「逐模块等价」，故 `make vet` = 对 `model/common/core/app` 逐模块 `go vet ./...`，只查本模块代码，当前 **0 错**。

## 9. 未验证项（Partial 原因）

1. **真实 GitHub Actions 绿灯**：未 push、未跑真 PR。无外网（决策门已批：worker 只写 workflow + 本地等价复现）；同组命令本机跑通/跑挂各一次已在 §3/§4/§5 留证。需人 push 后回看。
2. `check-tracked-imports.sh` 主检在当前 HEAD 为红（`thunder/logs` 未跟踪），**依赖 h1 先入库**；Task DAG 里 h3 deps = h1,h2，按序合并后该阻塞项转绿。本轮只做本地预演，不修改/不提交该脚本。
3. integration 是 `continue-on-error: true` allow-fail 起步；转阻塞不在本 Slice。
4. 未做发布流水线；未加 golangci-lint（用现成 gofmt+vet）；未起 HTTP 服务（`:8080`/`:8081` 未动）。

## 10. h1 / h2 口径

- **h1**：`scripts/check-tracked-imports.sh` **预热用，随 h1 入库**。h1 还会把 `thunder/logs/logs.go` 纳入跟踪；实测该文件 gofmt-clean，h1 入库后不会新增 gofmt 欠账、也不会破坏 §7 的 ratchet。
- **h2**：`app/go.mod` 当前仍声明 gorm `v1.25.12`，但 workspace MVS 实际已解析到 `gorm v1.31.2`（`go vet all` 的模块缓存路径可见 `gorm.io/gorm@v1.31.2`）。故 **h2 合并后声明会变、构建行为不变**：本轮全部验收按当前 HEAD 声明跑，未碰 `go.mod`/`go.sum`（黑名单）。

## 11. git diff 白名单证明

```
.github/workflows/ci.yml                                              (新)
.harness/rules/slice-contract.md                                      (改 1 行)
Makefile                                                             (改：.PHONY + 4 target)
plans/slices/harden-h3-ci-and-test-entrypoints.md                     (改：仅 Context Card；文件内另有 coordinator 的决策门 diff，非本 worker)
scripts/check-brief.sh                                                (新)
scripts/check-tracked-imports.sh                                      (h1 预热产物，未改/未提交)
scripts/gofmt-baseline.txt                                            (新)
scripts/gofmt-check.sh                                                (新)
scripts/test-integration.sh                                           (新)
docs/acceptance-records/2026-11-05-harden-h3-ci-and-test-entrypoints.md (本 record)
```
- `git diff --name-only -- '*.go'` = 空；`git diff` 无 `go.mod`/`go.sum`；无 `app/**` / `core/**` / `common/**` / `model/**` 业务改动。未 commit/push。

## 12. 已知坑 / 下一 Slice 建议

- 作者机器默认 `go` 1.27.3；本地跑本组命令务必 `PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local`（等价 CI 的 1.26.x），否则 grpc/x-net 漂移会让 `go build all` / `go test` 假红。
- zsh 下 `$P` 变量不会词分割、`PIPESTATUS` 与 bash 不同 —— 本机复现命令用显式 `env` 前缀、`>file; echo $?` 取退出码。
- macOS `/bin/bash` 3.2：`set -u` 下 `"$s」"` 会把全角括号字节吞进变量名，须写 `"${s}"`。
- 一次性容器就绪存在竞态，已硬化；仍建议 integration 保持 allow-fail 若干次再转阻塞。
- 建议后续：格式化那 10 个欠账文件并清空 baseline；h1/h2 合并后按序回看 Actions；integration 连续绿后单独提「转阻塞」。
