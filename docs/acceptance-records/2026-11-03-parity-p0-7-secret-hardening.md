# Acceptance Record — parity-p0-7-secret-hardening

- 日期：2026-11-03
- Slice：`plans/slices/parity-p0-7-secret-hardening.md`
- Epic：`plans/epics/xhs-parity-p0-correctness.md` §P0-7
- 主交付物：`common/config/secrets.go`（prod 硬校验）+ compose 必填变量 + 三级 `check-secrets.sh` + 文档四篇

## status: **Degraded**

note（为什么不是一句 Migrated）：

1. **三道闸门都落地且实测过**：compose `${JWT_SECRET:?}`（缺变量直接失败）→ `config.Load` prod 硬校验
   （占位值/长度/字符集/DB 口令/service_role 来源）→ lab 启动 `Warn` 清单 → `check-secrets.sh` 弱值/短值/高熵扫描。
2. **prod 校验链路只验到「拒绝启动」这一半**：本机没有真实 prod 依赖（PG `sslmode=require`、Redis、RabbitMQ 均非生产），
   所以「强密钥 + 强 DB 口令能一路启动成功」只验到 config 层放行（`strong_exit` 停在下一步 DB 校验 = 预期的第二道拒绝），
   没有跑通完整 prod 启动。
3. **`.env` 里的弱值本身没改**：Brief 明确「不改 `.env` 已入库的历史弱值内容本身（属破坏性动作，先与人确认）」，
   本片只加校验与文档。轮换需与 Flutter 端同步发版（见 startup-guide §13 轮换）。
4. **`Makefile:69,72,93` 仍硬编码 `postgres:postgres`**（H11 原文点名过）：不在本片白名单，
   且属本机 dev 迁移工具链 → 归 P0-1「迁移命令收口」一并处理，此处显式登记为残留。
5. **`REDIS_PASSWORD` 未纳入校验**：Brief 的三条 prod 口径里没有它，凭空加会误伤内网免密部署 → 不做（不是漏做）。

## 硬门槛实测

### ① `JWT_SECRET` 未设置 → compose 层失败并给可读错误 —— **PASS**

```
$ env -u JWT_SECRET docker compose -f docker/docker-compose.yml -f docker/docker-compose.lan.yml config
error while interpolating services.app.environment.JWT_SECRET: required variable JWT_SECRET is missing a value:
JWT_SECRET 必填：本地见 .env.example，部署须注入随机串（不再有兜底默认值）
exit=1
```

正向对照（lan 流程不被打断）：`set -a; source .env.lan; docker compose -f … -f … config` → exit 0。
改动点：`docker/docker-compose.yml:94,146`、`docker/docker-compose.lan.yml:34,54`（共 4 处，全部去掉了 `:-change-me-in-production` 兜底）。

### ② 显式给占位值 → 进程拒绝启动并打印原因 —— **PASS（三种值）**

`APP_ENV=prod` + 二进制实跑（`./scripts/load-env.sh env APP_ENV=prod JWT_SECRET=… /tmp/p07-api`）：

| 注入值 | 退出码 | 打印 |
|--------|--------|------|
| `change-me-in-production` | 1 | `启动失败: prod 启动校验失败：JWT_SECRET 仍是占位默认值，请换成 ≥32 位随机串` |
| `str0ng-random-value-…-Ijkl`（合格） | 1 | `prod 启动校验失败：DATABASE_PASSWORD 为已知默认值或长度不足 8`（继续往下拦，说明 JWT 这关已过） |
| `""` | 1 | `启动失败: jwt.secret 不能为空`（`configs/config.yaml` 兜底已删为空串） |

错误信息一律不回显密钥值（单测 `TestValidateSecretsProdRejectsWeakAndShort` 断言 `err` 不含原值）。

### ③ `make check-secrets` 对四类样例全部报错 —— **PASS**

样例文件 `/tmp/p07-samples.env`（非 lab 文件 → 硬失败）：

```
❌ 含 Supabase service_role 密钥，GitHub 会拒绝推送            ← 回归旧能力
❌ /tmp/p07-samples.env:JWT_SECRET 使用占位默认值
❌ /tmp/p07-samples.env:DATABASE_PASSWORD 使用占位默认值
❌ /tmp/p07-samples.env:APP_SECRET 长度 8 < 16，过短
❌ /tmp/p07-samples.env:SUPABASE_SERVICE_ROLE_KEY 疑似真实密钥（高熵，长度 22）
❌ /tmp/p07-samples.env:WECHAT_APP_SECRET 疑似真实密钥（高熵，长度 32）
❌ /tmp/p07-samples.env:JWT_SECRET 含空白字符（多为复制粘贴或引号写法出错）
exit=1
```

仓库自身：默认 `./scripts/check-secrets.sh` → 2 条 ⚠️（`.env` 的 lab 占位值）+ exit 0（不拦日常推送）；
`CHECK_SECRETS_STRICT=1` → 同样两条升级为 `❌ … 2 条占位弱值在部署口径下不允许`，exit 1。

判定口径的边界（防误报）：`*.example` 模板整体豁免弱值/熵判定（占位值正是模板该有的内容）；
`SUPABASE_ANON_KEY` / 各平台 `*_APP_ID` / `*_APP_KEY` 属设计上公开的标识位，不参与高熵判定；
yaml 只走 service_role 一条规则（`dev_test_password` 一类不撞 env 变量名规则）。

### ④ 现有登录链路不回归 —— **PASS**

`APP_ENV=dev AUTH_PROVIDER=local` 起 API 于 `:8083`：

```
health=200
POST /api/v1/user/phone/otp/verify (13400000000 / 123456) → login_code=0, token_len=291
WARN api/main.go:96 密钥体检：存在联调可用、生产必改的配置
     {"findings":["JWT_SECRET: 命中已知占位默认值","DATABASE_PASSWORD: 命中已知默认值"],"app_env":"dev"}
```

即：dev 不被新校验打断，但把风险摆到启动日志上。

## 机器验证

> Brief 的 `go build all` 被先于本片存在的 `thunder/logs` 缺目录卡死 → 分模块 build 替代。

| 命令 | 结果 |
|------|------|
| `go -C common build ./...` / `go -C core build ./...` | exit 0 |
| `go -C app build ./...`、`go -C app vet ./...` | exit 0 |
| `go -C common test ./config/... -count=1` | ok（含新增 `secrets_test.go` 5 个用例 / prod 8 子例） |
| `go -C model test ./...` | ok |
| `go -C app test ./... -count=1` | 全绿（无 FAIL 包；含 `common/config` 改动后的下游回归） |
| `./scripts/check-secrets.sh`（默认 / STRICT / 传样例） | 如上三档 |
| `docker compose … config`（缺失 / 正常两态） | 如上 |
| `bash --version` | 脚本在 **bash 3.2.57**（macOS 默认 `/usr/bin/env bash`）下跑通，未用 `${v,,}` 等 bash4 语法 |

## 改动清单（全在白名单内）

| 文件 | 改动 |
|------|------|
| `common/config/secrets.go` | 新增：弱值字典、`secretShapeProblem`、`SecretFindings()`、`validateSecrets()` |
| `common/config/secrets_test.go` | 新增：prod 8 子例 + DB 默认口令 + service_role 三来源 + findings 不泄值 + `Load` prod 拒绝/dev 放行 |
| `common/config/config.go` | `applyLocalEnvDefaults` 返回来源文件名；`Config.serviceRoleSource`；`Load` 末尾挂 `validateSecrets` |
| `configs/config.yaml` | `jwt.secret: ""`（删占位兜底）+ 注释说明密钥只能由 env 注入 |
| `docker/docker-compose.yml` / `docker-compose.lan.yml` | 4 处 `${JWT_SECRET:-…}` → `${JWT_SECRET:?…}` |
| `scripts/check-secrets.sh` | 三级判定（service_role 基线不动 + 弱值/短值/高熵/空白）+ lab 文件提示 + STRICT 档 + 支持显式传文件 |
| `app/cmd/api/main.go` | 仅启动期 `Warn` 打印 `SecretFindings()`（Brief 白名单允许「仅挂启动校验」） |
| `.env.example` / `.env.lan.example` | 「本地可弱、部署必改」+ compose 必填说明 |
| `docs/startup-guide.md` | 新增 §13 密钥（三道闸门表 + check-secrets 口径 + 轮换）；更正 `jwt.secret 不能为空` FAQ；env 表 `JWT_SECRET` 改「必填」 |
| `docs/project-detail.md` | H11 回写「已缓解」+ 残留点名 |
| `docs/README.md` | 检查项补「本地可弱、部署必改」与 STRICT 口径 |

黑名单未触碰：`common/jwt/`（算法与 token 结构原样）、`app/internal/usecase/`、`model/`、`.env` 内容本身。

## 待办

1. `Makefile:69,72,93` 的 `postgres:postgres` → 归 P0-1 迁移命令收口（同一处改动最省）。
2. prod 全链路（强密钥 + 强 DB 口令 → 一路启动成功）需要在真实 prod 依赖上补一次；本片只验到 config 层放行。
3. `configs/supabase.env` 的 anon key 是否为生产项目凭据（H11 关联的 Q6）仍需人确认——anon key 设计上可公开，
   但要确认没有把 service_role 误写进该文件（脚本现在会硬拦）。
