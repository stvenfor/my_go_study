# Acceptance — parity-p0-4-remove-fake-review

- Date: 2026-11-03
- Slice: `plans/slices/parity-p0-4-remove-fake-review.md`
- Status: **Degraded**
  - note：「超时=过审」的假审核已彻底删除，人工放行链路（含权限校验）本机真库跑通；
    仍缺机审 provider（Deferred → P2-2）、驳回态（无 `rejected` 状态，只有放行/不改）、
    批量放行界面（本片只给 SQL + 单条接口）。故不能记 Migrated。
- Mode: **Verified（本机）** / Partial（第三方与客户端侧未跑）

## 机跑

命令均在 `my_go_study/` 根目录执行，`export GOPROXY=https://goproxy.cn,direct`（原因见「环境坑」）。

| 命令 | 结果 |
|------|------|
| `go -C app build ./...` | exit 0（app 模块全部包） |
| `go -C app build -o /tmp/p04-api ./cmd/api` | exit 0 |
| `go -C app build ./cmd/worker` | exit 0 |
| `go vet ./common/config/...` | exit 0 |
| `go -C app test ./internal/...` | 34 个含测试包全 `ok`，无 FAIL |
| `go test ./common/... ./model/...` | `common/config`、`common/auth`、`common/jwt`、`common/metrics`、`model/entity` 全 `ok` |
| `git grep -n "reviewDelay\|ReviewDelay\|dueForApprove" -- app common core model` | **空**（谓词与配置项无残留） |
| `go build all` | **失败** —— 失败点不在本片，见「预存在的仓库问题」 |

## 真库人证（本机 Docker PG16 + Redis，:8081 `reject_until_manual`）

实例：`./scripts/load-env.sh env SERVER_PORT=8081 GRPC_ENABLED=false QUEUE_ENABLED=false
AUTH_PROVIDER=local SHORT_VIDEO_REVIEW_MODE=reject_until_manual /tmp/p04-api`；`GET /health` → `{"status":"ok"}`。
账号：dev 测试号 `13400000000`（作者 A）/ `13400000001`（他人 B），OTP `123456`。

| # | 步骤 | 实测 |
|---|------|------|
| 1 | `POST /api/v1/short-videos {"title":"P0-4 反证视频"}` | 201，`status=reviewing` |
| 2 | 作者 `GET /:id` | 200，`status=reviewing` |
| 3 | 他人 `GET /:id` | **404 / code 10004** |
| 4 | 他人 `GET ?scope=discovery` | `total=0`（不含该条） |
| 5 | **等待 65s**（越过原 30s 阈值）后查库 | `select status, approved_at` → **`0` / `NULL`** |
| 6 | 作者 `GET ?scope=user` | `total=1`，`status=reviewing`（作者自查仍可见） |
| 7 | 无角色用户 `POST /:id/review {"action":"approve"}` | **403 / code 10003「无权限」**，库里 `status` 仍 `0` |
| 8 | 给 B 授 `platform_admin` 后 `POST /:id/review` | 200，`status=normal`，`approved_at=2026-11-03T06:12:41Z` |
| 9 | 放行后他人 `scope=discovery` / `GET /:id` | `total=1` 含该条且 `status=normal`；详情 200 |
| 10 | 重复放行 | 200，`approved_at` 前后完全相同（幂等，不刷新） |
| 11 | `{"action":"reject"}` | 400「action 仅支持 approve」 |

演示模式（第二个实例 `:8082`，`SHORT_VIDEO_REVIEW_MODE=auto_allow_local_only`）：
新建视频后 B 立刻在 `scope=discovery` 看到该条 → Flutter 联调不被阻塞（验收第 5 条）。

## 已交付

- 删除「按时间自动置 approved」的全部路径：`dueForApprove`、`reviewDelay` 字段、
  `ShortVideoListQuery.Now/ReviewDelay`、`Stats(..., now, delay)`、`repo_postgres.go` 里
  `status=normal OR (reviewing AND created_at<=now-delay)` 的泄漏条件（原查询确实**对外泄漏未过审内容**）。
- `common/config`：`short_video.review_delay_seconds` → `short_video.review_mode`
  （`reject_until_manual` 默认 | `auto_allow_local_only`，后者仅在 `APP_ENV` 非 prod 生效）。
- 人工放行入口：`POST /api/v1/short-videos/:id/review`（body `{"action":"approve"}`），
  权限复用现成平台能力码 `role.assign_platform`（等价「平台管理员」），不新建权限码/迁移。
- 可见性收紧：非作者只认 `status=normal`；作者自查仍带 `reviewing`。
- 配置落盘：`configs/config.yaml`（默认 reject）、`config.dev.yaml` / `config.lan.yaml`（auto_allow 演示）。
- 文档同步：`docs/short-videos-api.md`（§2 约定/配置/状态机、§5 路由表 + 新增 §6.8、§8 客户端要点、§10 Q5 标记被推翻）、
  `docs/feishu/go-deep-dive/{02,08}`。
- 测试：`service_test.go` 新增 3 条（时间永不放行、放行需平台管理员、演示模式放行），
  `router` 基线 50→51 条（新增 review 路由，注明出处）。

## 历史数据处置（改完不会「一夜内容消失」）

上线本片后，库里既有的 `status=0` 行不再会自己变可见，需人工批量放行。dev 库上验证过的形态：

```sql
-- 先看规模
SELECT count(*) FROM wys_short_videos WHERE status = 0 AND deleted_at IS NULL;
-- 批量放行（保留原 created_at，approved_at 记放行时刻）
UPDATE wys_short_videos
   SET status = 1, approved_at = now(), updated_at = now()
 WHERE status = 0 AND deleted_at IS NULL;
```

放行范围/节奏属运营决策，本文件只给可执行语句；若需要留痕可先 `SELECT id` 导出清单再 UPDATE。
（Brief 白名单不含 `app/cmd/`，故未新增命令行工具；HTTP 单条接口 + 上述 SQL 覆盖需求。）

## 文件白名单偏差

| 文件 | 为何越界 |
|------|----------|
| `app/internal/delivery/http/router/domain_routes_test.go` | 新增第 9 条路由必然打破「与切片前逐条一致」基线；改基线并注明来源 |
| `app/internal/testsupport/access_stub.go` | 需要可配的平台能力码；新增字段是**附加式**（未配置时行为不变），避免把 17 方法桩复制回域内 |
| `docs/feishu/go-deep-dive/{02,08}` | 这两篇逐行描述被删除的 `reviewDelay` 行为，不改就成了假真相 |
| `app/internal/short_video/service.go` 引用 `app/internal/access` | 沿用 `home`/汽车四域既有先例（直接注入 `*access.Service`），未再造端口接口 |

## 环境坑（后续片务必知悉）

1. **`proxy.golang.org` 在本机不通**，且 `github.com/cloudwego/eino@v0.9.21` 不在本地 module 缓存
   → 任何 `go build/test` 直接网络超时失败。本轮所有验证走 `GOPROXY=https://goproxy.cn,direct`
   （已确认可达，未改任何仓库文件、未改全局 `go env`）。
2. **`go build all` 在 HEAD 上本就是红的**（与本片无关）：`thunder/req|midd|upload` import
   `github.com/mszlu521/thunder/logs`，而工作树里没有 `thunder/logs/`，`git ls-files thunder/logs` 为空
   → 复现：`go -C thunder build ./...`。所以 AGENTS.md 里的 `go build all` 这道门目前不可用，
   本片改用 `go -C app build ./...`（覆盖 app 模块全部包）作为等价机检。
3. `make lan-up` 遗留的 `my_go_study_app` / `my_go_study_worker` 容器正在 crash-loop（连不上 RabbitMQ）；
   本轮验证没动它们。PG/Redis 容器（`my_go_study_postgres`/`my_go_study_redis`）被我拉起后**仍在运行**，
   需要时用 `docker stop my_go_study_postgres my_go_study_redis` 收回。
4. 该 dev 库被本轮验证写入：2 个 OTP 用户、17 条短视频、B 账号一条 `platform_admin` 授予。需要干净复测可清 `user_role` / `wys_short_videos`。

## 未验证 / 风险

- Flutter 端未跑（客户端对 `status` 新值无感知风险低：未引入新状态值）。
- `APP_ENV=prod` 下 auto_allow 失效仅有代码层保证（`IsLabAppEnv("prod")=false`），未真起 prod 实例。
- 未接机审、无「驳回态」；`reviewing` 内容目前只能放行或作者自删。
- 权限选择 `role.assign_platform` 是「复用现成码」的取舍：语义上平台管理员 ≈ 内容审核者，
  若后续要求独立审核角色，需新增权限码 + 迁移（P2-2 一并处理）。
