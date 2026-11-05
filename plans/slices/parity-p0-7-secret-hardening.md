# Slice — parity-p0-7-secret-hardening

## Slice Brief
- SOURCE_MODULE: 默认值兜底的密钥配置 + 只认 `sb_secret_*` 的扫描
- TARGET_MODULE: 启动期密钥强度校验 + 通用密钥扫描
- Source entry: `docker-compose.yml:94`、`:146`（`JWT_SECRET: ${JWT_SECRET:-change-me-in-production}`）、`common/config/config.go:335`（硬编码兜底密钥）、`scripts/check-secrets.sh:13`（只匹配 `sb_secret_*` / `SUPABASE_SERVICE_ROLE_KEY=sb_`）、`.env`（已入库，含弱默认 `JWT_SECRET`/`DATABASE_PASSWORD`，见 `project-detail.md` H11）
- Target entry: `validateSecrets()` 挂在启动校验入口（与 P0-1 的启动退出策略共用）
- 本轮 ONLY:
  - compose 去掉兜底默认值 → 改必需变量 `${JWT_SECRET:?...}`，缺失时 compose 直接失败
  - 删 `config.go:335` 硬编码兜底；非 local 环境缺失即拒绝启动
  - `validateSecrets()`：`APP_ENV != local` 时校验 JWT secret 长度/字符集、DB 密码非默认、`SUPABASE_SERVICE_ROLE_KEY` 只允许来自 `.env.local`
  - `scripts/check-secrets.sh` 增强：默认值字典（`change-me-in-production`、`postgres`、`secret`、`123456`）+ 短串告警 + 高熵检测
  - 文档：`.env.example` / `docs/startup-guide.md` 明确「本地可弱、部署必改」；回写 `project-detail.md` H11 为已缓解
- 不做:
  - 不接 Vault/KMS/配置中心（差距文档 §8 明确列为当前不做）
  - 不改 JWT 签名算法与 token 结构（会影响 Flutter 端）
  - 不动 dev 固定 OTP（归 P0-5）
  - 不改 `.env` 已入库的历史弱值内容本身（只加校验与文档说明；若需轮换另立任务并同步客户端）
- 验收: 见 Epic §P0-7 全部 6 条；硬门槛：①`JWT_SECRET` 缺失时 `make lan-up` 在 compose 层失败 ②显式给 `change-me-in-production` 时进程拒绝启动 ③`make check-secrets` 对 4 类样例全部报错
- 文件白名单:
  - scripts/check-secrets.sh
  - docker/docker-compose.yml
  - docker/docker-compose*.yml
  - common/config/config.go
  - .env.example
  - .env.lan.example
  - configs/*.example
  - docs/
  - app/cmd/api/main.go（仅挂启动校验）
- 文件黑名单:
  - common/jwt/（不改算法）
  - app/internal/usecase/
  - model/
  - .env（不写入真实密钥；如必须改需先与人确认）
- 验证命令:
  - `go build all`
  - `go -C common test ./config/...`
  - `make check-secrets`
  - 负例：`JWT_SECRET=change-me-in-production APP_ENV=production go -C app run ./cmd/api` → 拒绝启动
  - 负例：`unset JWT_SECRET && make lan-up` → compose 报错
  - 正例：`make run`（local）→ `curl -sS http://127.0.0.1:8080/health` 200 + `POST /api/v1/user/login` 正常（AGENTS.md 检查项）
- 证据: docs/acceptance-records/2026-11-<date>-parity-p0-7-secret-hardening.md

## Context Card — parity-p0-7-secret-hardening
- 已完成:
  - `common/config/secrets.go`（新增）：弱值字典 + `secretShapeProblem`（未设置/含空白或引号/全同字符/长度不足）
    + `SecretFindings()`（lab 启动清单，weak 优先于 shape，**不回显值**）+ `validateSecrets()`（仅非 lab 生效）
  - `common/config/config.go`：`applyLocalEnvDefaults` 返回来源文件名 → `Config.serviceRoleSource`，
    `Load` 末尾挂 `validateSecrets`；`configs/config.yaml` 的 `jwt.secret` 兜底改空串
  - compose 4 处 `JWT_SECRET` 由 `:-change-me-in-production` → `${JWT_SECRET:?…}`（`docker-compose.yml:94,146`、
    `docker-compose.lan.yml:34,54`）：缺变量在**插值期**就失败，容器一个都不会起
  - `scripts/check-secrets.sh` 重写为三级判定：service_role 基线不动 + 弱值字典 / 短值(<16) / 高熵(≥20 且 ≥3 字符类) /
    含空白；lab 文件（`.env`）走 ⚠️，`CHECK_SECRETS_STRICT=1` 升级为 ❌；支持显式传文件
  - `app/cmd/api/main.go` 启动期 `Warn` 打印 `SecretFindings()`（Brief 允许「仅挂启动校验」）
  - 文档：`docs/startup-guide.md` 新增 §13 密钥（三道闸门 + 扫描口径 + 轮换）、`.env.example`/`.env.lan.example`
    「本地可弱、部署必改」、`project-detail.md` H11 回写已缓解、`docs/README.md` 检查项
  - 硬门槛 ①②③④ 全部实机 PASS（②覆盖占位值/合格值/空串三种注入）
- 未做/Deferred:
  - `.env` 内已入库的弱值本身未改（Brief 明确「需先与人确认」）→ 验收条 4「`git grep change-me` 无配置生效点」保持未勾
  - `Makefile:69,72,93` 硬编码 `postgres:postgres` → 超本片白名单，归 P0-1 迁移命令收口一并处理
  - prod 全链路（强密钥一路启动成功）只验到 config 层放行，后续拒绝点落在 DB → 待真实 prod 依赖补跑
  - `REDIS_PASSWORD` 刻意不校验（Brief 三条口径里没有它，硬加会误伤内网免密部署）
  - Vault/KMS/配置中心、JWT 算法与 token 结构、dev 固定 OTP → 均在 Brief「不做」内
- 关键文件: `common/config/{secrets.go,secrets_test.go,config.go}`、`configs/config.yaml`、
  `scripts/check-secrets.sh`、`docker/docker-compose{,.lan}.yml`、`app/cmd/api/main.go`、`docs/startup-guide.md` §13
- harness: post ok? — 是（record `docs/acceptance-records/2026-11-03-parity-p0-7-secret-hardening.md`，status=Degraded + 5 条 note；
  改动全在白名单内，黑名单 `common/jwt/`、`app/internal/usecase/`、`model/`、`.env` 内容未触碰）
- 下一 Slice 建议: 进 P0-1 `parity-p0-1-migration-gate`（**同动 `app/cmd/api/main.go` 与 `common/config/config.go`，需串行合入**；
  P0-1 顺手收口 `Makefile` 的 `postgres:postgres` 残留）；其后 P0-3 → P0-2（P0-2 依赖 P0-3 的事务边界）
- 已知坑:
  - AGENTS.md：`SUPABASE_SERVICE_ROLE_KEY` 仅 `.env.local`；推送前 `make check-secrets`
  - `.env` 是**有意入库**的（历史 commit `afbc29f`），本片只加校验；如需轮换密钥或改文件内容，属破坏性动作，**先与人确认**
  - 校验逻辑本身不能打印密钥值（只报「长度不足/命中默认值」），否则日志成新泄漏面 —— 单测里断言 `err` 不含原值
  - compose 用 `${VAR:?}` 会让未配 `.env.lan` 的同学直接起不来 → 需在 `docs/startup-guide.md` 与 `.env.lan.example` 同步说明
  - 模块边界：`common/config` 不得 import `app`/`core`（AGENTS.md 约束 10）
  - **prod 校验只能挂在非 lab 环境**：compose 里 `APP_ENV=lan` 属 lab，若在 lab 也硬校验会直接打断 `make lan-up`；
    「lab 只 Warn、prod 拒启动」是刻意分工，别在 lab 加严
  - `check-secrets.sh` 必须兼容 macOS 默认 **bash 3.2.57**：`${v,,}` 会语法报错 → 用 `tr '[:upper:]' '[:lower:]'`
  - **`*.example` 模板要整体豁免弱值/高熵判定**：占位值本就是模板该有的内容，否则 `make check-secrets` 天天红
  - 高熵检测需排除**设计上公开**的标识（`SUPABASE_ANON_KEY` / `*_APP_ID` / `*_APP_KEY`）；yaml 只走 service_role 一条规则
  - 错误信息里「含空白」要在高熵之前判定，否则带空格的粘贴错误会被报成「疑似真实密钥」
  - `APP_ENV=production` 在本仓库根本起不来（缺 `configs/config.production.yaml`），验证 prod 闸门要用 `APP_ENV=prod`
