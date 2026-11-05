# Slice — parity-p0-4-remove-fake-review

## Slice Brief
- SOURCE_MODULE: `app/internal/short_video` 审核状态机
- TARGET_MODULE: 默认拒绝 / 人工放行（机审留待 P2-2）
- Source entry: `short_video/service.go:52-58`（`reviewDelay` 默认 30s）、`:339-340`（「超时即判通过」谓词）、`:160`（入库置 `reviewing`）、`:200`、`:295`（Stats/列表按 delay 计算状态）
- Target entry: `review.mode = reject_until_manual`（默认）| `auto_allow_local_only`（仅 `APP_ENV=local`）
- 本轮 ONLY:
  - 删除超时自动通过谓词及其全部调用点；`reviewDelay` 语义改为「最长待审」或彻底移除（推荐移除）
  - 放行只走人工入口：最小 `POST /api/v1/short-videos/{id}/review`（带角色校验，复用现成 `accountGate`/role 机制），或先提供 `app/cmd/` 手工命令
  - 可见性过滤：未过审内容对非作者不可见（先核 `short_video/repo_postgres.go` 现状是否已泄漏）
  - 配置与文档同步（`configs/*.yaml`、`docs/`）
- 不做:
  - 不接机审 provider（P2-2）
  - 不做社区内容审核（P2-2）
  - 不新建 `wys_content_reviews`（P2-2 建，本片只改状态语义）
  - 不动 `community` 域
- 验收: 见 Epic §P0-4 全部 6 条；核心反证：**新上传视频等待 > 原 30s 后仍为 `reviewing`**；`git grep reviewDelay` 无「按时间判通过」残留
- 文件白名单:
  - app/internal/short_video/
  - configs/
  - common/config/config.go
  - app/cmd/api/main.go（仅构造传参变化）
  - docs/
- 文件黑名单:
  - app/internal/community/
  - model/entity/short_video.go（除状态常量，勿改字段）
  - core/queue/
- 验证命令:
  - `go build all`
  - `go -C app test ./internal/short_video/...`
  - `make run` → 造一条短视频 → 等 60s → 查状态仍 `reviewing`（curl 或 SQL）
  - 双账号：作者可见、他人列表不可见
  - 人工放行后他人可见
  - `APP_ENV=local` 演示模式一键可用（客户端不阻塞）
- 证据: docs/acceptance-records/2026-11-<date>-parity-p0-4-remove-fake-review.md

## Context Card — parity-p0-4-remove-fake-review
- 已完成: 见 `docs/acceptance-records/2026-11-03-parity-p0-4-remove-fake-review.md`
  （谓词/配置项删除、`review_mode` 两档、`POST /:id/review` 平台管理员放行、可见性收紧、
  3 条新单测 + 本机真库 11 步人证全过；状态 **Degraded**：无机审、无驳回态）
- 未做/Deferred: 机审 provider 与 `rejected` 态（P2-2）；批量放行仅给 SQL，未做 `cmd/`（白名单不含）
- 关键文件: `app/internal/short_video/service.go`（`visibleTo` / `Approve` / `lazyApprove*`）、
  `common/config/config.go`（`ShortVideoConfig.ReviewMode` / `ShortVideoAutoAllowUnreviewed`）、
  `app/internal/short_video/repo_postgres.go`（查询侧不再按时间泄漏）、`configs/config{,.dev,.lan}.yaml`
- harness: post ok?（机检：`go -C app build ./...` + `go -C app test ./internal/...` 全绿；
  `go build all` 因 thunder 预存在问题不可用，见 record「环境坑」2）
- 下一 Slice 建议: `parity-p0-5-fake-impl-inventory`（本片结果登记进清单）
- 已知坑:
  - 演示/联调会因此变「麻烦」：必须保留 `APP_ENV=local` 的 auto_allow 分支，否则 Flutter 侧看不到内容（AGENTS.md 联调检查项）
  - 审核语义 ≠ `home` 域店务审批状态机（`migrations/20260922160000_home_todo.up.sql:56`），勿复用其表与角色
  - 若现有数据大量处于 `reviewing`，改完会「一夜内容消失」——需在 record 里给出历史数据处置方案（人工批量放行命令）
  - `short_video` 媒体 URL 缺省填 picsum/公开 mp4（`service.go:32-39`）不在本片修（归 P0-5 登记 + P2-1 实现）
