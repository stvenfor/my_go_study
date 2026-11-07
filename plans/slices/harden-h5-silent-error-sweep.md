# Slice — harden-h5-silent-error-sweep

## Slice Brief
- SOURCE_MODULE: `app`（`internal/delivery/worker`、`internal/usecase`、`internal/access`）+ `common/metrics`
- TARGET_MODULE: 同上
- Source entry: 实测 **5 处**（口径已核到行）
  1. `delivery/worker/handler.go:216` —— 入队失败后 `_, _ = h.redis.Del(ctx, dedupKey)`：删去重键失败 = 该用户该时段**永久跳过**推送，且无痕迹
  2. `usecase/realtime_push_usecase.go:168` —— `_, _ = u.offline.NotifyUser(...)`：离线推送失败不影响返回，用户收不到且无人知
  3. `access/service.go:102` —— `_, _ = s.storeGroupSync.EnsureStoreMembership(...)`：加店群失败但成员已入库，数据不一致
  4. `usecase/profile_usecase.go:82` —— 同上形状（切店成功、拉群失败静默）
  5. `usecase/im_group_usecase.go:163` —— `CreateGroup` 失败被 `_ = err` 吞掉，注释称「群可能已存在」，但网络/凭据错误走同一分支
  - 注意：P0-2/P0-3 曾把 `membership` 的 `_, _ = Credit(...)` 已改为事务回滚（`docs/acceptance-records/2026-11-04-membership-money-tx-followup.md`），**那部分不在本 Slice 范围**
- Target entry: 每条静默失败都有「日志 + `degraded_path_total{name=...}` 指标 + 可观测后果」；会丢数的那条升级为硬失败并交给既有重试/死信机制
- 本轮 ONLY: 改这 5 处的错误处置与留痕；复用现成的 `common/metrics.DegradedPathsTotal`（`common/metrics/metrics.go:36-44`，标签名与 `docs/placeholder-inventory.md` 条目一一对应）
- 不做:
  - **不新增 logger 字段/构造函数参数**（`app/internal/usecase/*.go` 目前无 zap logger；注入要改 `cmd/api/main.go`，与 h8 冲突 → 走指标 + 已有 `h.log`，其余留 Deferred）
  - 不引入 outbox / 重试框架（P2-8）
  - 不改业务成功语义（除第 1 条明确升级为失败）
  - 不动 `payment`/`membership`
- 验收:
  - [ ] 逐条改动表（改前行为 / 改后行为 / 可观测差异）落在 acceptance-record，风险 R4 要求每条都能回答「会不会把 best-effort 变成 500」
  - [ ] `worker/handler.go:216`：删键失败改为 `h.log.Error` + 指标，并且**该错误要参与返回**（宁可整任务重投，也不要静默留下坏的去重键）；配单测断言「Del 失败时 handler 返回 error」
  - [ ] 其余 4 处：保持降级返回，但都 `DegradedPathsTotal.WithLabelValues("<name>").Inc()`；name 与 `docs/placeholder-inventory.md` 条目名一致
  - [ ] `/metrics` 增量人证：触发一次 `realtime_push_usecase.go:168` 降级，`curl -s http://127.0.0.1:8081/metrics | grep degraded_path_total` 计数 +1（端口沿用本轮约定的 **8081**）
  - [ ] `git grep -nE '_, _ = |^\s*_ = err' -- app/internal | grep -v _test` → 只剩**逐条豁免登记**的条目（或 0）
  - [ ] `go -C app test ./internal/... -count=1` 绿；`go build all` 绿
- 文件白名单:
  - `app/internal/delivery/worker/handler.go`
  - `app/internal/usecase/realtime_push_usecase.go`
  - `app/internal/usecase/profile_usecase.go`
  - `app/internal/usecase/im_group_usecase.go`
  - `app/internal/access/service.go`
  - `common/metrics/metrics.go`（仅当需要新增/规范 name 常量）
  - 上述包的 `*_test.go`
  - `docs/placeholder-inventory.md`（追加式）
  - `docs/acceptance-records/2026-11-05-harden-h5-silent-error-sweep.md`
  - `plans/slices/harden-h5-silent-error-sweep.md`（仅 Context Card）
- 文件黑名单:
  - `app/cmd/api/main.go`（h8 独占）
  - `app/internal/{payment,membership,mall}/**`
  - `configs/*.yaml`、`docker/`（h7 范围）
  - `**/go.mod`、`**/go.sum`
- 验证命令:
  - `go -C app test ./internal/delivery/worker/ ./internal/usecase/ ./internal/access/ -count=1`
  - `go build all`
  - `git grep -nE '_, _ = |^\s*_ = err' -- app/internal | grep -v _test`
  - `curl -s http://127.0.0.1:8081/metrics | grep degraded_path_total`
- 证据: `docs/acceptance-records/2026-11-05-harden-h5-silent-error-sweep.md`

## Context Card — harden-h5-silent-error-sweep
- 已完成: 5 处静默吞错处置（第 1 条硬失败 + 单测，其余 4 条降级 + 指标）；5 个 name 追加登记 inventory 条目 22–26；
  `/metrics` 8081 curl 人证 `offline_push_failed` 0→1→2；`go build all` + 三包 test 绿。详见
  `docs/acceptance-records/2026-11-05-harden-h5-silent-error-sweep.md`。
- 未做/Deferred: 4 条降级的日志（无 logger，不新增构造参数）；完整 `cmd/api` 端到端人证（本轮用同 Registry/Handler 探针）；
  其余 4 处单测；按融云错误码分类 `CreateGroup` 失败；去重键自愈（P0-6）。
- 关键文件: `app/internal/delivery/worker/handler.go`（`cleanupDedupAfterEnqueueError`）、
  `app/internal/usecase/{realtime_push_usecase,profile_usecase,im_group_usecase}.go`、`app/internal/access/service.go`、
  `common/metrics/metrics.go`（只读复用 `DegradedPathsTotal`）、`docs/placeholder-inventory.md`（条目 22–26）。
- harness: post ok? `go version`=1.26.4（默认 1.27.3 编 grpc 会红，按共同前提用 `/usr/local/go/bin/go`）；
  `go build all`=0；三包 test=ok；`git grep` 残留 3 条已逐条豁免登记；不 commit/push、不 `git add` 未跟踪构建前置。
- 下一 Slice 建议: usecase 层注入 logger 补 4 条降级日志；`im_group` 按错误码分类；P0-6 去重键自愈/DLQ。
- 已知坑: name 用字面量、靠 inventory 对齐（name 增多后应提常量）；`DegradedPathsTotal` 是进程级全局 CounterVec，
  单测用 before/after 差值断言，勿重置全局。
