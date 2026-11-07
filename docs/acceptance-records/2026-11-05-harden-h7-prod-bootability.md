# Acceptance — harden-h7-prod-bootability

- Date: 2026-11-07
- Slice: `plans/slices/harden-h7-prod-bootability.md` · taskId `task_f10481ca877d`
- Mode: **Degraded**
- Note（Degraded 原因，按 coordinator 要求列残留）：核心代码、单测、本机真机 `curl` 人证全部完成；
  **未跑完整 `make lan-up` + Prometheus UI `Targets=UP` 端到端人证**（本机已有同名的
  `my_go_study_postgres/redis` 容器在跑，整栈 `--build` 会重建/冲突；且缺 `.env.lan`，
  新建它属改本机配置，未获授权故不做）。Prometheus 抓取保活改以「网段实测 + 单测 +
  compose render」三项机证替代，见下。另见「残留」节。
- harness: 本机实跑；`go` 用 `/usr/local/go/bin/go` = **go1.26.4 darwin/amd64**
  （默认 go1.27.3 的 `go build all` 会因 grpc v1.71.0 `http2.TrailerPrefix` 报红，非本轮 bug）
- 自证：`pwd=/Users/qpy/orca/workspaces/my_go_study/harden-h7` ✅ · `HEAD=e288553` ✅

## 一、改动清单（均在白名单内）

| 文件 | 改动 |
|------|------|
| `common/config/config.go` | 新增 `normalizeAppEnv`（小写/trim + `production`→`prod`）、`knownLabAppEnv`；`Load` 归一化 env，正式/未知/拼错环境缺覆盖文件**拒绝启动并点名文件**，不再 Warn 后回退 debug 基础配置 |
| `app/internal/delivery/http/router/router.go` | `/metrics` 挂载加 `metricsAccessGuard`（只认 `RemoteAddr`，loopback 放行，局域网 403），`METRICS_ALLOW_CIDRS` 显式放行 Docker 内网抓取 |
| `docker/docker-compose.yml` | app 服务注入 `METRICS_ALLOW_CIDRS: ${METRICS_ALLOW_CIDRS:-172.16.0.0/12,10.0.0.0/8}`（保住 Prometheus `app:8080/metrics`） |
| `docker/docker-compose.lan.yml` | 头部端口策略注释补 `/metrics` 收口说明 |
| `common/config/app_env_test.go`（新增） | 环境名归一化 / prod 覆盖值 / 缺失拒绝 / 未知拒绝 / staging 回退 5 组 |
| `app/internal/delivery/http/router/metrics_access_test.go`（新增） | 守卫单测（loopback / LAN / XFF 伪造 / Docker 内网 / 真实 Setup 路由） |
| `docs/observability-lan.md`、`docs/startup-guide.md`、`docs/lan-backend-host.md`、`docs/placeholder-inventory.md` | 同步更正（含删除「`APP_ENV=production` 无法启动」的错误说明） |
| 本文件、`plans/slices/…md` Context Card | 证据与收口 |

未改：`go.mod` / `go.sum` / `go.work`（黑名单，`git diff --name-only | grep go.mod` 为空）、
`app/cmd/api/main.go`（h8）、`controller/**`（h4）、`worker/**`/`usecase/**`（h5）、`.env` / `.env.local`。
未 commit / push / rm -rf。

## 二、机跑（全部 go1.26.4）

```
/usr/local/go/bin/go version            -> go version go1.26.4 darwin/amd64
/usr/local/go/bin/go build all          -> exit 0
/usr/local/go/bin/go -C common test ./... -count=1                        -> ok all
/usr/local/go/bin/go -C app test ./... -count=1                           -> ok all
/usr/local/go/bin/go -C app test ./internal/delivery/http/... -count=1    -> ok（Brief 指定）
```

配置值可跑证明（`go -C common test ./config/ -run '…' -v`）：

```
APP_ENV="prod"        -> AppEnv="prod" server.mode="release" database.sslmode="require" database.auto_migrate=false
APP_ENV="production"  -> AppEnv="prod" server.mode="release" database.sslmode="require" database.auto_migrate=false
APP_ENV="Production"  -> AppEnv="prod" server.mode="release" database.sslmode="require" database.auto_migrate=false
APP_ENV=" production "-> AppEnv="prod" server.mode="release" database.sslmode="require" database.auto_migrate=false
TestLoadProdRefusesMissingOverlay / TestLoadUnknownEnvRefusesMissingOverlay / TestLoadKnownLabEnvStillFallsBack -> PASS
```

compose render（基础 + lan overlay 叠加）：

```
JWT_SECRET=… REALTIME_PUBLIC_WS_HOST=… docker compose -f docker/docker-compose.yml -f docker/docker-compose.lan.yml config -q -> render OK
  METRICS_ALLOW_CIDRS: 172.16.0.0/12,10.0.0.0/8
```

## 三、负例证明（真二进制启动，非仅单测）

隔离临时目录只放 `configs/config.yaml`（**不放** `config.prod.yaml`），
`go -C app build -o /tmp/h7-api ./cmd/api` 后从该目录启动：

```
APP_ENV=production /tmp/h7-api
 -> 启动失败: APP_ENV="prod" 缺少环境覆盖文件 configs/config.prod.yaml：拒绝以基础 config.yaml（debug / sslmode=disable）启动
APP_ENV=prod /tmp/h7-api
 -> 启动失败: APP_ENV="prod" 缺少环境覆盖文件 configs/config.prod.yaml：拒绝以基础 config.yaml（debug / sslmode=disable）启动
APP_ENV=prodution /tmp/h7-api   # 故意拼错
 -> 启动失败: APP_ENV="prodution" 缺少环境覆盖文件 configs/config.prodution.yaml：拒绝以基础 config.yaml（debug / sslmode=disable）启动
```

即「缺覆盖文件 = 明确失败且点名文件」，**不再有 Warn 后按 debug 继续跑**。

## 四、对照证明（真实 `go -C app run ./cmd/api`，端口 8081）

> 本地约束：本机 PG 无 TLS、无 migrations 版本、`DATABASE_PASSWORD` 为弱占位值，故以下启动
> **仅为本机人证**加了 env 覆盖：`DATABASE_SSLMODE=disable`、
> `DATABASE_SCHEMA_VERSION_CHECK=false`、`QUEUE/SCHEDULER/GRPC_ENABLED=false`，并连一次性
> 临时 PG（5433，强口令）。`config.prod.yaml` 本身的 `sslmode=require` 值由上面单测证明。
>
> 另注：`APP_ENV=prod ./scripts/load-env.sh …` 实测会被 `.env` 的 `APP_ENV=dev` 覆盖成 dev；
> 必须写成 `./scripts/load-env.sh env APP_ENV=prod …`（已写进 startup-guide §3.2）。

`APP_ENV=prod`（真实 overlay）启动日志与 curl：

```
schemaboot: 启动期 DDL 已关闭：schema 由发布流程 migrations/ 管理 {"app_env": "prod"}
main.go:728 跳过演示种子（APP_ENV 非联调环境）
main.go:545 HTTP 服务启动 {"port": 8081, "env": "prod"}
grep '使用基础 configs/config.yaml'  -> 无（无回退行）
grep -c '\[GIN-debug\]'             -> 0   （release 模式；对照 dev 启动日志为 52 行）
curl http://127.0.0.1:8081/health   -> 200
```

`APP_ENV=production`（归一化）同样：

```
health=200；无回退行；[GIN-debug] 0 行
schemaboot {"app_env": "prod"}（配置侧已归一化）
main.go:545 HTTP 服务启动 {"port": 8081, "env": "production"}（main.go 仍打印原始 APP_ENV）
```

## 五、`/metrics` 人证（真实服务，端口 8081）

```
curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8081/metrics        -> 200
curl -s -o /dev/null -w '%{http_code}' http://192.168.2.56:8081/metrics     -> 403
curl -s http://127.0.0.1:8081/metrics | head -4 -> # HELP es_transaction_index_failures_total …（真指标体）
```

dev / prod / production 三次启动结果一致。服务日志：

```
{"status": 200, "path": "/metrics", "ip": "127.0.0.1"}
{"status": 403, "path": "/metrics", "ip": "192.168.2.56"}
```

## 六、Prometheus 抓取保活（替代完整 lan-up 的三项机证）

1. **网段实测**：临时容器 `-p 18080:8080`，从宿主机分别 `curl 127.0.0.1:18080` 与
   `curl 192.168.2.56:18080`，容器内 `RemoteAddr` 均为 **`192.168.65.1`**（docker-proxy 网关）；
   而 compose 网桥 `docker_default=172.18.0.0/16`、`bridge=172.17.0.0/16`，属 `172.16.0.0/12`。
   → Prometheus 容器（`app:8080/metrics`，源 172.18.x）被放行；经端口映射的局域网/宿主机
   访问（源 192.168.65.1）被拒。
2. **单测** `TestSetupMetricsAllowsDockerInternalScrape`：真实 `Setup` 路由下
   `172.18.0.5` / `10.1.2.3` → 200，`192.168.65.1` → 403。
3. **compose render**：`METRICS_ALLOW_CIDRS` 正确注入 app 服务；`prometheus.yml` 的
   `targets: ["app:8080"]`、4 条 payment 告警规则均未改动。

`curl 127.0.0.1`（loopback）始终 200 → 「Prometheus 抓本机口」未被弄坏。

## 七、文档同步

- `docs/observability-lan.md`：`/metrics` 收口章节 + Docker 下改用 `docker exec` 的冒烟说明
- `docs/startup-guide.md`：§3.1 归一化与「缺失即拒绝」、§3.2 `APP_ENV` 行 + `.env` 覆盖坑、
  §9 观测表注、§12 生产部署 `production` 归一化
- `docs/lan-backend-host.md`：§3.3 端口策略 + §6 验收清单新增第 8 行
- `docs/placeholder-inventory.md`：更正「`APP_ENV=production` 无法启动」的过期注释 + 追加 h7 记录

## 八、残留（Degraded 依据）

1. **未跑完整 `make lan-up` + Prometheus UI `Targets=UP` 端到端人证**：本机已有同名
   `my_go_study_postgres/redis` 容器在跑，整栈 `--build` 会重建冲突；且缺 `.env.lan`
   （新建属改本机配置，未获授权）。以第六节三项机证替代。
2. **Docker 部署下宿主机 `curl 127.0.0.1:8080/metrics` 变 403**（对端是 docker-proxy 网关
   192.168.65.1，与局域网流量无法区分）。已在 docs 给出 `docker exec my_go_study_app curl
   http://127.0.0.1:8080/metrics` 的等价冒烟方式；本机 `make run` 进程不受影响。
   API 的 `/health` 仍对局域网开放（业务需要），未改 `ports` 映射。
3. **`METRICS_ALLOW_CIDRS` 默认值假定 Docker 网桥落在 `172.16.0.0/12` / `10.0.0.0/8`**；
   自定义网段需部署方覆盖该变量（compose 已留 `${METRICS_ALLOW_CIDRS:-…}` 入口）。
4. **`main.go` 启动日志仍打印原始 `APP_ENV` 字符串**（`production`），与 `cfg.AppEnv=prod`
   不一致；`main.go` 在 h8 白名单外，本轮未改（仅文案，不影响行为）。
5. **本机 prod 人证依赖 env 覆盖**（见第四节）：`sslmode=require` / `schema_version_check`
   在本机无 TLS PG、无迁移库的条件下无法原样启动；值由单测证明。真实生产仍须注入
   `JWT_SECRET` / `DATABASE_PASSWORD` 等密钥（既有设计，非新增 env 文件依赖）。
6. **已知 lab 环境（`staging` 等）缺覆盖文件仍回退基础配置**：为保留既有 P0-1 行为与
   `TestLoadSchemaGateDefaults`，未做成「全部未知环境一律拒绝」；仅正式/未知/拼错名拒绝。
   若后续要连 lab 也收紧，属扩大范围，需新 Slice。

## 九、未验证项与主要风险

- 未验证：完整 `lan-up` 全栈镜像重建、Prometheus Targets=UP、Grafana dashboard、飞书告警链路。
- 主要风险：若生产 Docker 网桥不在默认放行网段，Prometheus 抓取会被 403（需覆盖
  `METRICS_ALLOW_CIDRS`）；`alerts.yml` 的 `APITargetDown` 会先于本轮文档告警。
