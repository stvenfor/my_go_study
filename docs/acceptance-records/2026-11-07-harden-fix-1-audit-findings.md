# Acceptance — harden-fix-1 审计缺陷修复（2026-11-07）

## 触发

对已合并的 harden-h1…h10（`main@6989590`）做只读审计，3 个槽并行、不扩槽：

| 槽 | agent | 范围 | 结论 |
|---|---|---|---|
| A | CodeBuddy（`hy4free high`） | h1–h3 | h1/h2 无缺陷；h3 1×P1 + 3×P2 |
| B | CodeBuddy（`hy4free high`） | h4–h7 | h4/h6/h6b 无缺陷；h5 1×P1；P2 共 6 条 |
| C | Cline（`nemotron-3.5-super-vl`） | h8–h10 | 1×P2（settle.go 注释残留） |

只读约束：不改文件、不 commit、不 DDL、不写业务库 `my_go_study`。coordinator 逐条复核证据后才进入修复。

## 缺陷与处置

| ID | 级别 | 位置 | 处置 | status |
|---|---|---|---|---|
| A-P1 | P1 | `scripts/test-integration.sh`、`.github/workflows/ci.yml` | 补齐 `P06B_PG_DBNAME` / `P06B_AM_DBNAME` / `P08_PG_DBNAME`；本机与 CI 幂等建库；对 `p06b_it` 应用 `migrations/*.up.sql` | Migrated |
| A-P2a | P2 | `scripts/test-integration.sh` | external 模式 `pg_isready`；`-v`；`FAIL`/`SKIP` 即失败；`PASS ≥ 21` | Migrated |
| A-P2b | P2 | `Makefile` | `check-go` 守卫（`go1.26.*`），默认 1.27.3 给中文提示而非编译报错 | Migrated |
| A-P2c | P2 | `scripts/check-brief.sh` | 六节改为 `## Slice Brief` 内字段头结构校验；0 slice 即 FAIL；self-test 加「正文假放行」反例；补 h3/h10 Brief | Migrated |
| B-P1 | P1 | `app/internal/access/service.go:129` | 离店踢群失败由 `_ =` 改为 `degraded_path_total{access_store_group_removal_failed}`；inventory 条目 35；h5 验收 grep 改 `^[[:space:]]*_ = ` | Migrated |
| B-P2a | P2 | `app/internal/delivery/http/controller/jpush_controller.go` | `RegisterDevice` 丢弃客户端 alias，强制 `user.ID`；新增落库断言 | Migrated |
| B-P2b | P2 | `app/cmd/api/main.go:545` | 启动日志改用 `cfg.AppEnv` | Migrated |
| B-P2c | P2 | `.env.example`、`.env.lan.example` | 补 `METRICS_ALLOW_CIDRS` 注释行 | Migrated |
| B-P2d | P2 | `docs/observability-lan.md` | 补 Linux userland-proxy 条件性说明，去掉「端口映射必 403」的无条件结论 | Migrated |
| C-P2 | P2 | `app/internal/payment/settle.go:105-108` | 更新过时注释（membership 已自注册 + 支持 txdb） | Migrated |

保留 Deferred（未改代码/未实机复跑）：

- h7「未知 env 名 + 存在覆盖文件」仍可启动（原验收已登记 Deferred）。
- h6b 迁移门槛通过时无 `NOTICE` 留痕；属已文档化设计取舍，不改已合并迁移。
- `/metrics` 默认 CIDR 在 Linux Docker 下可能放行 `172.17.0.1`：本轮只补文档口径，未收窄默认值、未 Linux 真机复跑。
- h5 修正 grep 后额外命中的 42 处既有 `_ =`（白名单外、非本轮引入）逐条未改，已在 h5 验收记录登记。

## coordinator 复跑证据（原始输出摘要）

```bash
PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local make test-integration
# go_exit=0 PASS=21（期望 ≥ 21） FAIL=0 SKIP=0
# OK: integration 门禁通过

PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local make lint        # exit 0
make lint                                                       # exit 2
# 需要 Go 1.26.x（当前 go1.27.3）。请用 PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local make <target>

./scripts/check-brief.sh --self-test                            # [self-test] PASS（4/4）
PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local make agent-pre   # OK: 29 个 slice

PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local go -C app test ./internal/delivery/http/controller/... ./internal/payment/...
# ok controller / ok payment

PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local go build all     # exit 0
bash -n scripts/test-integration.sh scripts/check-brief.sh      # exit 0
```

## 口径裁决

- `test-integration.sh` 为达成「0 SKIP」使用 `-run` 只跑 21 个集成用例：**接受**。该入口契约本就是「跑 `-tags=integration` 用例」，用例名从 `*_integration_test.go` 动态提取（新增用例自动纳入）；普通单元测试由 `make test` 覆盖。已在脚本输出中显式打印 `run <21 个集成用例>`。
- 未跑真实 GitHub Actions（无外网、未 push）；CI 结论来自 workflow 静态核对 + 本机等价命令。

## 提交

- `ce30e72` fix(ci): 补齐集成测试 5 个 scratch 库并幂等建库
- `b597c97` fix(access): 离店踢群失败打 degraded 指标并修正 h5 验收 grep
- `acb70dd` merge: harden-fix-1 P1
- `d23a10b` fix(ci): 集成门禁防空绿并加 Go 1.26 工具链守卫
- `0859f6f` fix(harness): check-brief 改六节结构校验并补齐 h3/h10 Brief
- `ee705b0` fix(h4/h7/h8): 强制会话 alias、日志用归一化 env、补模板与口径
- `5c022cf` merge: harden-fix-1 P2
