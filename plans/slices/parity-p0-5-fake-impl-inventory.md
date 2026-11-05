# Slice — parity-p0-5-fake-impl-inventory

## Slice Brief
- SOURCE_MODULE: 全仓「返回成功但没做事」的代码路径
- TARGET_MODULE: `docs/placeholder-inventory.md`（五态登记）+ 未接路径改显式 501/未开通
- Source entry（首批必录，均已在差距文档核实）:
  - `membership/handler.go:152-155`（华为 IAP 回调直接 `accepted:true`）
  - `delivery/worker/handler.go:169-177`（SMS 任务只打日志）
  - `phone_otp_usecase.go:31-42`（OTP 发送 no-op）+ dev 固定码 `13400000000/123456`
  - `short_video/service.go:32-39`、`community/service.go:30-37`（mock 图/视频 URL）
  - `repository/llm/mock_stream_provider.go` + `main.go:608-616`（无 key 回退 mock）
  - `cs_agent_usecase.go:98,153`（客服不调 LLM，模板拼接+伪流式）
  - `analytics_usecase.go:37-49`（演示种子）
  - `community/repo_postgres.go:371-434`、`short_video/repo_seed.go`、`mall/repo_seed.go`
  - `local_auth_usecase.go:252,326`（占位邮箱 `wx_/hw_` 域名）
  - JPush 占位分支（见 `project-detail.md` §10.4）
- Target entry: 清单 + 语义修正 + `APP_ENV` 守卫
- 本轮 ONLY:
  - 建 `docs/placeholder-inventory.md`，每条含：文件:行、status（`Migrated/Degraded/Placeholder/Deferred/Removed`）、note、影响面、解除条件、对应 Slice
  - **未真实实现却返回成功**的分支 → 改 `501 Not Implemented` / 业务码「未开通」并说明
  - dev 固定 OTP、seed 自动执行、mock 媒体填充 → 加 `APP_ENV=local|dev` 守卫（复用 P0-7 的启动校验入口）
  - `Degraded` 类保留可用，但必须有 note + 日志 + 指标计数可见
- 不做:
  - 不去实现任何被登记为 `Deferred/Placeholder` 的能力（那是别的 Slice）
  - 不删 mock provider 代码（本地/测试需要，只是不让它在生产静默生效）
  - 不做扫描自动化门禁（可选，成本高则 Deferred）
- 验收: 见 Epic §P0-5 全部 7 条；硬门槛：①`APP_ENV=production` 下固定码 `123456` **不能登录** ②华为回调不再返回成功语义 ③清单与代码抽查 3 条一致
- 文件白名单:
  - docs/placeholder-inventory.md
  - app/internal/membership/handler.go
  - app/internal/delivery/worker/handler.go
  - app/internal/usecase/phone_otp_usecase.go
  - app/internal/usecase/local_auth_usecase.go
  - app/internal/usecase/cs_agent_usecase.go
  - app/internal/usecase/analytics_usecase.go
  - app/internal/short_video/service.go
  - app/internal/community/service.go
  - app/internal/repository/llm/
  - configs/
  - common/config/config.go
  - docs/
- 文件黑名单:
  - core/jpush/（只登记，不改实现）
  - migrations/
  - thunder/
- 验证命令:
  - `go build all`
  - `go -C app test ./internal/... -run 'OTP|Auth|Membership'`（受影响面定向测试）
  - `APP_ENV=production make run` → 用固定码调 `POST /api/v1/user/phone/otp/verify` 应失败
  - curl 华为回调 stub 路由 → 501/未开通
  - `make run`（local）→ 固定码仍可登录（不阻断联调）
- 证据: docs/acceptance-records/2026-11-<date>-parity-p0-5-fake-impl-inventory.md

## Context Card — parity-p0-5-fake-impl-inventory
- 已完成:
  - `docs/placeholder-inventory.md` 落盘：17 条（华为回调 / SMS / OTP send / dev 固定码 / 短视频+社区 mock 媒体 /
    LLM mock 回退 / 客服模板 / analytics 种子 / 10 处启动 seed / 占位邮箱 / JPush skip / ConfirmBuyout /
    P0-4 已删的假审核 / 机审 / 无调用方的 EnqueueSendSMS / 扫描门禁）+ 4 条指向其它 Slice
  - 语义修正：华为回调 200`accepted:true` → **501 + 业务码 10005**（新增 `CodeNotImplemented`）
  - SMS stub：`Info` → `Warn(status=placeholder)`，补单测锁「ACK 丢弃 + 恰 1 条 Warn」
  - `APP_ENV` 守卫：10 处启动 seed 收口到 `main.go:seedIfLab`；mock 媒体填充收口到
    `config.IsLabAppEnv(appEnv)`（`short_video.NewService(...,appEnv)` / `community.NewService(...,appEnv)`）；
    dev 固定码本已由 `DevBypassEnabled` 门住，补 local provider 的 prod 拒绝单测
  - Degraded 可见性：`common/metrics` 新增 `DegradedPathsTotal`，`llm_mock` / `cs_agent_template` 已抓取到计数
  - 实机验收（dev :8081 + prod :8082 双进程并存）：15 项证据见 acceptance-record
- 未做/Deferred:
  - 扫描 `TODO|FIXME|mock|stub` 的 CI 门禁 → Deferred（清单条目 17，噪声成本高于价值）
  - worker 进程指标（SMS）→ 无 `/metrics` 端点，只给日志；JPush skip 计数 → 改 `jpush_usecase.go` 超白名单
  - `ConfirmBuyout` 不验支付、`EnqueueSendSMS` 无调用方 → 只登记，实现归 P0-2 / 短信 Slice
  - 未实现任何被登记为 Deferred/Placeholder 的能力（按 Brief「不做」）
- 关键文件:
  - `docs/placeholder-inventory.md`、`docs/acceptance-records/2026-11-03-parity-p0-5-fake-impl-inventory.md`
  - `app/cmd/api/main.go`（`seedIfLab` / `labSeeds` / SSE 回退日志 / 两个 NewService 传 appEnv）
  - `app/internal/{short_video,community}/service.go`（`demoMedia` 守卫）、`app/internal/membership/handler.go`
  - `app/internal/delivery/http/dto/response/common.go`（`CodeNotImplemented=10005`）
  - `common/metrics/metrics.go`（`DegradedPathsTotal`）、`app/internal/delivery/worker/handler.go`
- harness: post ok? 是 —— record 标 `Degraded` + note（不声称 Migrated）；白名单偏离逐条列出；
  抽查 3 条 status 与行为一致（实机 + 单测双证）；Brief 里两条验证命令不可执行（见下坑）已点名替换
- 下一 Slice 建议: `parity-p0-6-mq-dlx-retry`
- 已知坑:
  - **Brief 的 `APP_ENV=production make run` 当前起不来**：`config.Load` 需要 `configs/config.production.yaml`，
    仓库只有 `config.{,dev,lan,prod}.yaml`；本片用 `APP_ENV=prod` + 临时配置目录验证（`IsLabAppEnv` 语义等价），
    环境名收敛归 P0-7/P0-1
  - **Brief 的 `go build all` 仍失败**（`thunder/logs` 目录缺失，先于本片存在）→ 替换为
    `go -C app build ./...` + `go -C common build ./...` + `go -C app build -o /tmp/x ./cmd/worker`
  - `proxy.golang.org` 不可达 → go 命令需 `GOPROXY=https://goproxy.cn,direct`
  - `my_go_study_app/worker` 容器 crash-loop，无法用 `make lan-up` 验证 prod 语义
  - `.harness/rules/status-enums.md`：`Degraded/Placeholder/Deferred` **必须**有 note；「能编译 / `/health` 通 ≠ Migrated」
  - `.harness/rules/no-silent-accept.md`：未跑验证命令不得勾验收
  - 改返回语义是**破坏性变更**，Flutter 侧若依赖当前 200 需跨仓同步（Program R3 / Q8）—— 本仓未做跨仓验证
  - dev 固定码是联调依赖（AGENTS.md 明写 `13400000000 + 123456`），别一刀切删——按环境守卫
  - 本 Slice 只改「语义」，别顺手补功能实现（最小改动原则）
  - 改 `NewService` 形参后要同步 `router/domain_routes_test.go` 与各域测试，否则 vet 编译失败
