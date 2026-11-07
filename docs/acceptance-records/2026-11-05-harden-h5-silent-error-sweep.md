# Acceptance Record — harden-h5-silent-error-sweep

- 日期：2026-11-07（执行日；文件名沿用 Slice 白名单指定的 `2026-11-05-...`）
- Slice：`plans/slices/harden-h5-silent-error-sweep.md`（taskId `task_af7088ce6598`）
- 自证：pwd=`/Users/qpy/orca/workspaces/my_go_study/harden-h5`，HEAD=`e288553`（`e2885539ee050d83d15665f0007d35c1a180d2a0`）✓
- 范围：`app/internal/delivery/worker`、`app/internal/usecase`、`app/internal/access` + 只读复用 `common/metrics`
- 未改：`cmd/api/main.go` / `payment` / `membership` / `mall` / `configs` / `docker` / `**/go.mod|go.sum`（遵守白/黑名单）；未 commit/push、未 `git add` 任何未跟踪构建前置

## status: **Degraded**

note：5 处静默吞错全部改为「可观测」——第 1 条（会丢数）升级为硬失败交既有重试/死信，其余 4 条按 R4
保持 best-effort 降级但打 `degraded_path_total` 指标。usecase 层无 zap logger，4 条的日志留 Deferred
（Slice 明确「不新增 logger 字段/构造参数」）；`/metrics` 人证用同 Registry/Handler 的临时探针替代完整
`cmd/api` 端到端。均见「诚实口径」。

## 逐条改动表（含 R4：会不会把 best-effort 变成 500）

| # | 文件:行（改前） | name（= inventory 条目） | 改前行为 | 改后行为 | 可观测差异 | R4：best-effort → 500？ |
|---|----------------|--------------------------|----------|----------|------------|--------------------------|
| 1 | `delivery/worker/handler.go:216` | `push_dedup_del_failed` | 入队失败后 `_, _ = Del` 回滚去重键；Del 也失败则键残留 → 该用户该时段永久跳过推送，无痕迹 | 抽 `cleanupDedupAfterEnqueueError`：Del 失败 `h.log.Error` + 指标 +1，且把 enqueue/del 两个 error 一起 `%w` 返回 | 从「静默丢一条定时推送」变为「Error 日志 + 指标 + 任务重投/死信可见」 | **是（刻意）**：Slice 唯一点名升级为失败的一条；它不经 HTTP，走 worker 重试/DLQ，不会把任何 HTTP 请求变 500 |
| 2 | `usecase/realtime_push_usecase.go:168` | `offline_push_failed` | `_, _ = offline.NotifyUser` 丢弃离线推送失败 | `if err != nil` 打指标 +1，`DeliverPush` 仍返回成功、WS 广播不变 | 从「用户收不到且无人知」变为 `/metrics` 可计数 | **否**：返回语义完全不变（仍 nil error），只多一次纯内存 `Inc()`，无 500 |
| 3 | `access/service.go:102` | `access_store_group_sync_failed` | `_, _ = EnsureStoreMembership` 丢弃拉群失败（成员已入库） | `if err != nil` 打指标 +1，`UpsertMember` 仍返回 member, nil | 从「数据不一致无痕迹」变为指标可见 | **否**：成员入库不回滚，无 500 |
| 4 | `usecase/profile_usecase.go:82` | `profile_store_group_sync_failed` | `_, _ = EnsureStoreMembership` 丢弃（切店成功、拉群失败） | `if err != nil` 打指标 +1，`SwitchStore` 仍返回 stats, nil | 同 #3 | **否**：同上 |
| 5 | `usecase/im_group_usecase.go:163` | `im_group_create_failed` | `CreateGroup` 失败被 `_ = err` 吞掉，注释称「群可能已存在」，网络/凭据错误走同一分支 | 该分支打指标 +1，仍继续 join + upsert | 从「真失败被当已存在」变为指标可计数（但不分类） | **否**：控制流不变，仍 join + upsert 返回，无 500 |

- 复用 `common/metrics.DegradedPathsTotal`（`common/metrics/metrics.go:39-45`），标签 `name` 取上表字面量；
  5 个 name 已**追加式**登记到 `docs/placeholder-inventory.md` 条目 23–27（状态速览表 + 末尾详细条目 + 语义变更摘要），
  与代码 `WithLabelValues("...")` 一一对应。`metrics.go` 未改（沿用 `llm_mock`/`cs_agent_template` 的字面量风格，最小改动）。

## 第 1 条：硬失败 + 单测

- `cleanupDedupAfterEnqueueError` 用 `fmt.Errorf("enqueue push for %s: %w; cleanup dedup key: %w", ...)`，
  两个 `%w`（Go 1.20+）让 `errors.Is` 对 enqueue 与 del 两个原始 error 都命中 → Del 失败**参与返回**。
- 新增 `app/internal/delivery/worker/handler_dedup_test.go::TestCleanupDedupAfterEnqueueError_DelFailurePropagates`：
  用 `redis.Options.Dialer` 强制 dial 失败制造**真实** Del 错误（零新依赖、不占端口），断言
  ① 返回非 nil error ② `errors.Is(err, enqueueErr)` ③ 文案含 `cleanup dedup key`（反映 Del 失败）
  ④ `degraded_path_total{push_dedup_del_failed}` +1 ⑤ 1 条 `Error` 日志。
  `handleBroadcastNotify` 入队失败分支 `return h.cleanupDedupAfterEnqueueError(...)`，故本方法返回 error 即 handler 返回 error。

## /metrics 增量人证（端口 8081）

- 工具链：`/usr/local/go/bin/go version` = **go1.26.4 darwin/amd64**（默认 `go` = 1.27.3 下 `go build all`
  会因 grpc v1.71.0 `http2.TrailerPrefix` 报红，非本改动问题，按共同前提用 1.26.4）。
- 临时探针 `app/cmd/h5probe/main.go`（**用完即删、未入库、未 git add**）在 `:8081` 挂生产
  `common/metrics.Handler()`（与 `cmd/api` 的 `/metrics` 同一个 Registry/Handler），`/trigger` 调真实
  `RealtimePushUsecase.DeliverPush` 并注入必然失败的 `OfflinePusher` 触发 168 行降级。
- 实跑（`curl -s http://127.0.0.1:8081/metrics | grep degraded_path_total`）：

```
=== BEFORE === (no degraded_path_total line yet = 0)
=== TRIGGER #1 === triggered offline push degrade
=== AFTER #1 === degraded_path_total{name="offline_push_failed"} 1
=== TRIGGER #2 === triggered offline push degrade
=== AFTER #2 === degraded_path_total{name="offline_push_failed"} 2
```

- 结论：每触发一次 168 行降级，`offline_push_failed` +1（0→1→2）。探针进程已停、文件已删，
  最终 `git status --short` 不含它。

## 验证

```bash
/usr/local/go/bin/go build all                                  # BUILD_EXIT=0
/usr/local/go/bin/go -C app test ./internal/delivery/worker/ ./internal/usecase/ ./internal/access/ -count=1
#   ok app/internal/delivery/worker / usecase / access
# \s 在 POSIX ERE 下不是标准转义（git grep -E 不识别），第二个分支恒不命中；改用 ^[[:space:]]*_ = ：
git grep -nE '_, _ = |^[[:space:]]*_ = ' -- app/internal | grep -v _test
```

（删探针前后各跑过一轮全工作区 `go build all` + 三包单测，均绿；grep 结果一致。）

## `git grep` 残留 —— 逐条豁免登记

验收 grep 原写作 `git grep -nE '_, _ = |^\s*_ = err' -- app/internal | grep -v _test`。
**`\s` 在 POSIX ERE 下不是标准转义**（`git grep -E` 不识别），第二个分支恒不命中，
因此 `_ = <fn>()` 形状（正是 `access/service.go` 离店踢群那一处）从未被验收扫到 ——
这就是「验收 grep 失效」的根因。harden-fix-1 把它改为 `^[[:space:]]*_ = `（见「验证」）。

改完 5 处后，原（失效）grep 仅剩 3 条（非 `_test`），均为**良性丢弃**且不在本 Slice 白名单，
显式登记为豁免清单：

| 文件:行 | 代码 | 豁免理由 |
|---------|------|----------|
| `app/internal/delivery/http/controller/cs_agent_controller.go:79` | `_, _ = c.Writer.Write([]byte(": keepalive\n\n"))` | SSE 心跳写；写失败即连接已断，循环由 `streamCtx.Done()` 收口，丢弃返回无后果。controller 不在白名单 |
| `app/internal/delivery/http/controller/sse_controller.go:94` | 同上 | 同上 |
| `app/internal/payment/service.go:444` | `_, _ = rand.Read(b)`（`crypto/rand`） | Go 1.24+ `crypto/rand.Read` 保证填满 `b` 且 err 恒 nil，丢弃返回值不影响结果；且 `payment/**` 在**黑名单** |

> 口径补记：这 3 条是 `_, _ = ` 形状，**不匹配**修正后的 `^[[:space:]]*_ = `。
> 修正后的模式会额外命中仓内其它文件里既有的 `_ = <fn>()`（共 42 行，
> `json.Unmarshal` / `ShouldBindJSON` / `conn.Close` / `_ = ctx` 等，分布在
> `new_car_follow` / `deal_invoice` / `delivery` / `ws` / `mall` / `community` /
> `usecase` / `membership` 等包），全部在 h5 白名单之外、非本轮引入，
> 属同类「良性丢弃」，不是 h5 回归；按「不在白名单、不擅自改」维持现状。

## 第 6 处修复（harden-fix-1，2026-11-07）

- 位置：`app/internal/access/service.go:129`（改前 `_ =` 在 `:127`）。
- 指标 name：`access_store_group_removal_failed`
  （`metrics.DegradedPathsTotal.WithLabelValues("access_store_group_removal_failed").Inc()`）。
- 改动：`RemoveMember` 内 `if s.storeGroupSync != nil` 分支由
  `_ = s.storeGroupSync.RemoveStoreMembership(...)` 改为
  `if err := ...; err != nil { 打指标 }`；**不回滚成员删除、非致命、返回语义不变**
  （与条目 25 `UpsertMember` 同形降级，方向相反）。
- 登记：`docs/placeholder-inventory.md` 条目 35（status `Degraded`）。
- 验收：`grep -rnE '^[[:space:]]*_ = ' app/internal --include='*.go' | grep -v _test`
  不再命中 `app/internal/access/service.go`；`go -C app test ./internal/access/... -count=1` 通过。

## 诚实口径 / 未验证项 / 风险

1. **4 条降级没有日志**：usecase/access 层无 zap logger，Slice 明确「不新增 logger 字段/构造参数（避免与 h8 改 main.go 冲突）」，
   故只打指标、日志留 Deferred。指标是无标签值的 Counter，不含 error 详情 → 能知道「发生了 N 次」但不知「为什么」。
2. **`/metrics` 人证是探针而非完整 `cmd/api` 端到端**：探针复用生产 `metrics.Handler()` / `Registry` 与真实
   `DeliverPush` 路径，但未起 `cmd/api`、未连 PG/Redis/RabbitMQ，也未经过 `SessionAuth` 路由。
   完整端到端（起 API 触发真实离线失败）留 Deferred；本轮只证「168 行降级 → 同一 Registry → `/metrics` +1」这条链。
3. **指标不分类**：条目 26（`im_group_create_failed`）无法区分「群已存在」与「网络/凭据真失败」，
   这是刻意保留的降级口径（Slice 未要求改控制流），需按融云错误码细分才能定向告警。
4. **第 1 条重投不消除坏键**：Del 失败 → 返回 error → 重投时 `SetNX` 因残留键仍会 skip。硬失败的价值是
   「不静默」+ 进 DLQ 告警，由人工/后续清理键；彻底自愈属 P0-6 重试/死信范畴，本轮不引入 outbox/重试框架。
5. **其余 4 处未配单测**：Slice 只点名第 1 条配单测；4 处的降级 + 指标由 `go build all` + 三包既有用例 +
   `/metrics` curl 人证（第 2 条）覆盖，未逐一写单元断言。
6. **未动 `common/metrics/metrics.go`**：name 用字面量（与现有 `llm_mock`/`cs_agent_template` 一致），
   靠 inventory 文档对齐；若后续 name 增多，建议提常量（Deferred）。

## 文件清单（相对白名单）

- 改：`app/internal/delivery/worker/handler.go`、`app/internal/usecase/{realtime_push_usecase,profile_usecase,im_group_usecase}.go`、
  `app/internal/access/service.go`、`docs/placeholder-inventory.md`、`plans/slices/harden-h5-silent-error-sweep.md`（Context Card）
- 增：`app/internal/delivery/worker/handler_dedup_test.go`、`docs/acceptance-records/2026-11-05-harden-h5-silent-error-sweep.md`
- 未动：`common/metrics/metrics.go`（不需要）、`cmd/api/main.go`、`payment`/`membership`/`mall`、`configs`/`docker`、`**/go.mod|go.sum`
- 临时（已删、未入库）：`app/cmd/h5probe/main.go`
