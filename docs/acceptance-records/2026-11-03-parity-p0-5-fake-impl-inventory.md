# Acceptance Record — parity-p0-5-fake-impl-inventory

- 日期：2026-11-03
- Slice：`plans/slices/parity-p0-5-fake-impl-inventory.md`
- Epic：`plans/epics/xhs-parity-p0-correctness.md` §P0-5
- 主交付物：`docs/placeholder-inventory.md`（17 条 + 指向其它 Slice 的 4 条）

## status: **Degraded**

note（为什么不是一句 Migrated）：

1. **清单本身是 Migrated 的**：17 条逐条给了 文件:行 / status / note / 影响面 / 解除条件 / Slice，
   且抽查 3 条（华为回调、SMS、OTP）行为与登记一致（实机 + 单测证据见下）。
2. **语义修正只做了「返回成功」里最危险的一条**：华为 IAP 回调 → 501/`10005`。
   `ConfirmBuyout`（条目 13）、`EnqueueSendSMS`（条目 16）按 Brief「不做：不去实现被登记为 Deferred/Placeholder
   的能力」保持原语义，只登记。
3. **Degraded 项的「指标可见」只覆盖 API 进程**：`degraded_path_total{llm_mock,cs_agent_template}` 已落地并抓取到；
   worker 进程（SMS）无 `/metrics` 端点，只给了 Warn 日志；JPush 的 skip 在 `jpush_usecase.go`（超白名单），
   保持响应体 `skipped/reason` 可见而未加指标。
4. **扫描自动化门禁 = Deferred**（条目 17，Brief 允许）。
5. **验收口径里写的 `APP_ENV=production` 目前起不来**：`config.Load` 要求 `configs/config.<env>.yaml` 存在，
   仓库只有 `config.yaml/dev/lan/prod`。本次用 `APP_ENV=prod` + 临时配置目录验证（`IsLabAppEnv` 对 prod/production
   同样返回 false，守卫等价）。这条本身作为运维坑登记，改态需 P0-7 一并处理。

## 机器验证（全部本机实跑）

> Brief 的两条验证命令按原文不可执行，已点名替换并在下表注明：`go build all`（thunder/logs 缺目录，先于本片失败）
> → 分模块 build；`APP_ENV=production make run`（缺 `configs/config.production.yaml`）→ `APP_ENV=prod` + 临时配置目录。

| 命令 | 结果 |
|------|------|
| `go -C app build ./...` | exit 0 |
| `go -C app build -o /tmp/p05-worker ./cmd/worker` | exit 0 |
| `go -C common build ./...` | exit 0 |
| `go -C app vet ./...` | exit 0 |
| `go -C app test ./internal/...` | exit 0（全绿，无 FAIL） |
| `go -C model test ./...` | exit 0 |
| `go -C app test ./internal/... -run 'OTP\|Auth\|Membership\|ShortVideo\|Community\|Media\|Seed'` | exit 0 |
| `go -C app test ./internal/delivery/worker/... -run SendSMS -v` | 2 PASS |

新增/改动的断言：

- `app/internal/usecase/phone_otp_usecase_test.go` → `TestPhoneOTP_ProductionRejectsDevFixedCode`
  同时覆盖 Supabase provider 与 **local provider**（dev/lan 实际链路）：`APP_ENV=production` 下
  `13400000000 + 123456` 的 send/verify 均 `ErrPhoneLoginNotAvailable`；dev 下 send 仍是 no-op 成功（占位事实）。
- `app/internal/short_video/service_test.go` → `TestShortVideo_ProdEnvRejectsMockMedia`
  prod 缺 URL → `ErrInvalid`；prod 带真实 URL → 正常入库且不被改写。
- `app/internal/community/service_test.go` → `TestRequiresRealMedia_gate`（7 组用例：image/video/none/空）。
- `app/internal/delivery/worker/handler_sms_test.go` → 占位分支必须 **ACK + 单条 Warn(status=placeholder)**；
  畸形载荷必须返回 error（不伪装成功）。

## 实机证据（两个进程并存：dev :8081 / prod :8082）

准备：`my_go_study_postgres`、`my_go_study_redis` healthy；`AUTH_PROVIDER=local QUEUE_ENABLED=false GRPC_ENABLED=false`；
prod 实例从 `/tmp/p05-prodroot`（`config.yaml` + 临时 `config.prod.yaml`，只改连接项）启动。
登录：dev 用 `otp/verify 13400000000/123456`；prod 用 `username=13400000000@dev.test.local` 邮箱口令。

| # | 检查 | 期望 | 实测 |
|---|------|------|------|
| 1 | dev 启动种子 | 10 条「已就绪」 | `main.go:608` ×10（短视频/首页×2/发票/二手车/新车/购车/商城/社区/analytics）✅ |
| 2 | prod 启动种子 | 全部跳过 | `跳过演示种子（APP_ENV 非联调环境）` ×10，日志内**无一条**种子 SQL 成功记录 ✅ |
| 3 | prod `otp/send` | 拒绝 | `400 {"code":10001,"message":"短信登录暂未开放，请使用邮箱登录"}` ✅ |
| 4 | prod `otp/verify 123456` | 拒绝（硬门槛①） | 同上 `400 10001`，未签发任何 token ✅ |
| 5 | dev `otp/send` | no-op 成功（占位事实） | `{"code":0,"data":{"ok":true}}` ✅ 与清单条目 3 一致 |
| 6 | `POST /membership/huawei/notifications` | 501（硬门槛②） | dev `http=501`；prod `http=501` `{"code":10005,...}` ✅ |
| 7 | prod 短视频缺媒体 | 拒绝填充 | `400 10001 short_video: invalid params: 缺少 video_url / cover_url，上传通道未接入（见 docs/placeholder-inventory.md）` ✅ |
| 8 | prod 短视频带真实 URL | 放行 | `201`，`video_url=https://cdn.example.com/a.mp4` 原样入库 ✅ |
| 9 | dev 短视频缺媒体 | 仍填充样例 | `201`，`bee.mp4` + `picsum sv_play_1` ✅ 联调未阻断 |
| 10 | prod 社区 image 缺 URL | 拒绝 | `400 10001 community: invalid params: 缺少真实图片/视频 URL…` ✅ |
| 11 | prod 社区 image 带真图 | 放行 | `201 images=["https://cdn.example.com/1.jpg"]` ✅ |
| 12 | dev SSE（mock） | 成功但计数 | SSE `model=mock-v1`；`/metrics` → `degraded_path_total{name="llm_mock"} 1` ✅ |
| 13 | dev 客服 Agent | 成功但计数 | `degraded_path_total{name="cs_agent_template"} 1` ✅ |
| 14 | prod SSE 启动日志 | 明确「未配置」 | `WARN SSE Provider=mock（未配置 LLM，回复为内置样例）{"status":"degraded","inventory":"docs/placeholder-inventory.md"}` ✅ |
| 15 | prod SSE 命中计数 | 与 dev 一致 | `degraded_path_total{name="llm_mock"} 1`（prod 进程）✅ |

清理：验证完 `pkill -f p05-api`，:8081/:8082 监听数 0。**未删任何数据**；prod 实例复用同一台 dev Postgres，
验证残留 2 条业务行（`p05-prod-real` 短视频、`p05-prod 带真图` 动态）+ dev 实例 1 短视频 / 1 动态。

## 清单 ↔ 代码 抽查（Brief 硬门槛③）

| 条目 | 登记 status | 实测行为 | 一致 |
|------|-------------|----------|------|
| 1 华为 IAP 回调 | `Placeholder`（已改 501） | 501 + code 10005（dev/prod 双实例） | ✅ |
| 2 SMS worker stub | `Placeholder`（Warn + ACK 丢弃） | 单测断言恰 1 条 Warn + `status=placeholder` + `err==nil` | ✅ |
| 3/4 手机号 OTP | `Placeholder`（发送 no-op）+ `Degraded`（固定码 lab-only） | dev send 返回 `ok:true` 且无短信；prod 双 provider 拒绝 | ✅ |
| 5/6 mock 媒体 | `Placeholder`（lab-only） | dev 填充 / prod 400（4 次实调） | ✅ |
| 7 LLM mock | `Degraded`（日志 + 指标） | 启动 Warn + `llm_mock` 计数 | ✅ |

## 文件白名单偏离（P0-5 Brief 未列，但改动必要）

| 文件 | 为什么必须改 |
|------|--------------|
| `app/cmd/api/main.go` | 种子守卫（条目 10 的「seed 自动执行」只能在这 10 个调用点收口）、SSE 回退日志、两个 NewService 传 `cfg.AppEnv` |
| `app/internal/delivery/http/dto/response/common.go` | 需要「未开通/未实现」业务码 `10005`（原枚举只有 10001~10004） |
| `common/metrics/metrics.go` | 新增 `DegradedPathsTotal`，给降级路径统一的指标口径 |
| `app/internal/{short_video,community}/…` 测试、`router/domain_routes_test.go` | 构造函数加 `appEnv` 形参，编译必需 |
| `app/internal/membership/router.go` | 注释仍写「MVP stub 便于联调」，与 501 新语义冲突（dual-truth） |
| `docs/{README,short-videos-api,community-posts-api}.md` | 文档描述默认媒体填充/占位语义，需与新守卫一致 |
| `app/internal/delivery/worker/handler_sms_test.go` | 新增（Brief 验证命令要求 SMS 有定向测试证据） |

未使用文件黑名单：`core/jpush/`、`migrations/`、`thunder/` 均未改动 ✅

## 已知坑（给后续 Slice）

- `config.Load` 缺 `config.<APP_ENV>.yaml` 直接启动失败：`APP_ENV=production` 无对应文件 ⇒ Brief 里的命令
  `APP_ENV=production make run` **当前不可执行**，需先有 `configs/config.production.yaml`（或在 P0-7 一并收敛环境名）。
- `proxy.golang.org` 不可达，所有 go 命令需 `GOPROXY=https://goproxy.cn,direct`。
- `go build all` 仍因 `github.com/mszlu521/thunder/logs` 目录缺失失败（先于本片存在），机器门槛以
  `go -C app build ./...` + `go -C common build ./...` + `go -C app build ./cmd/worker` 代替。
- `my_go_study_app` / `my_go_study_worker` 容器仍在 crash-loop（与本片无关，故验证走本机进程）。
- 仓库里存在游离二进制 `app/worker`（非本次产生），提交前需确认是否清理。

## 未验证 / 风险

- **Flutter 侧回归未跑**：华为回调改 501、发帖/发视频在 prod 改 400 属破坏性语义变更；本仓只验了 Go 侧。
  prod 语义目前不会影响 dev/lan 联调（已实测），但若 Flutter 有依赖 `accepted:true` 的逻辑需跨仓同步（Program R3/Q8）。
- 未跑 `make lan-up` 全栈验证（容器 crash-loop），prod 实例连的是**本机 dev Postgres**，
  所以「prod 环境」只是 APP_ENV 语义等价验证，不含真实 TLS/远端库差异。
- 条目 11（占位邮箱）、12（JPush skip）、13（ConfirmBuyout）、16（无调用方）为**登记不改**，
  行为仍与切片前一致；其中 13 的资金风险要靠 P0-2 收口。
- 清单行号会随改动漂移，后续 Slice 若动到这些文件需同步回写（当前 17 条已逐条按 HEAD 复核）。
