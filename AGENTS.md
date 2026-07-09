# my_go_study Agent 开发指南

供 AI Agent 与协作者快速理解 Go 后端结构与 Supabase 集成约定。

## 项目定位

- **框架**：Gin + GORM + PostgreSQL + Redis + Clean Architecture
- **模块路径**：`github.com/stvenfor/my_go_study`
- **默认端口**：`8080`
- **配对客户端**：Flutter `my_ai_project`

## 分层结构

```text
cmd/api/main.go                 # 依赖注入、启动
internal/
  delivery/http/
    handler/                    # 自建用户 HTTP（UserHandler）
    controller/                 # Supabase 业务 HTTP（Profile / Transaction）
    middleware/                 # JWT / SupabaseAuth / CORS / Logger
    router/                     # 按模块拆分路由
    dto/request|response/       # 请求/响应 DTO
  usecase/                      # 业务用例（含 SupabaseAuthUsecase）
  domain/entity|repository/     # 领域实体与仓储接口
  repository/
    postgres/                   # 本地 PostgreSQL
    supabase/                   # Supabase PostgREST
pkg/
  config/                       # Viper 配置
  supabase/                     # Supabase 客户端封装
  auth/                         # Token 校验（/auth/v1/user）
  jwt/                          # 自建 JWT（遗留路由）
```

## Supabase 集成要点

完整文档：[docs/supabase-integration.md](./docs/supabase-integration.md)

| 能力 | 入口 | Supabase 交互方式 |
|------|------|-------------------|
| 邮箱注册/登录 | `UserHandler` → `SupabaseAuthUsecase` | gotrue-go `Signup` / `SignInWithEmailPassword` |
| Token 校验 | `middleware.SupabaseAuth` | HTTP `GET /auth/v1/user` |
| Profile CRUD | `ProfileController` → `profile_repo` | PostgREST + `WithUserToken` |
| Transactions CRUD | `TransactionController` → `transaction_repo` | PostgREST + `user_id` 过滤 + RLS |

**启用条件**：`SUPABASE_URL` + `SUPABASE_ANON_KEY` 非空（`config.SupabaseConfig.Enabled()`）。

## 两套认证（勿混淆）

| 中间件 | 路由示例 | Token 类型 | 用户 ID 类型 |
|--------|----------|------------|--------------|
| `SupabaseAuth` | `/api/v1/me/*`、`/api/v1/transactions*` | Supabase access token | UUID string |
| `Auth`（JWT） | `/api/v1/user/list`、`/api/v1/user/profile` | Go 自建 JWT | `uint` |

当前 Flutter 登录返回 **Supabase token**，业务 API 走 `SupabaseAuth` 路由组。

## 配置与环境变量

```bash
cp .env.example .env
cp configs/supabase.env.example configs/supabase.env   # 团队 Supabase 常量（通常已入库）
cp .env.local.example .env.local                       # 本地 service_role
```

| 文件 | 入库 | 说明 |
|------|------|------|
| `configs/supabase.env` | 是 | `SUPABASE_URL`、`SUPABASE_ANON_KEY` 团队常量 |
| `.env` | 是 | 应用运行时（DB、Redis、JWT 等） |
| `.env.local` | 否 | `SUPABASE_SERVICE_ROLE_KEY` 私密密钥 |

推送前执行 `make check-secrets`，或启用 Git 钩子：

```bash
git config core.hooksPath .githooks   # 可选，本地一次配置
```

## 常用命令

```bash
make run                  # 启动 API
make test                 # 单元测试
make test-transactions    # transactions 联调
./scripts/check_transactions_rls.sh
```

## Agent 修改规范

1. **所有 `go` / `make` 命令在 `my_go_study` 目录内执行**，不要在父目录 `my_code_study` 执行
2. 新增 Supabase 表：entity → repository 接口 → `repository/supabase` 实现 → usecase → controller → router（挂 `SupabaseAuth`）
3. 认证错误在 `SupabaseAuthUsecase.mapSupabaseAuthError` 映射，HTTP 层在 `UserHandler.handleUsecaseError` 处理
4. PostgREST 请求必须用 `client.WithUserToken(accessToken)`，不可仅用 Admin 客户端绕过 RLS
5. `transactions` 查询必须带 `.Eq("user_id", userID)`，并确保 Supabase RLS 已迁移
6. 响应 DTO 放 `internal/delivery/http/dto/response/`，Flutter 兼容接口注意 snake_case
7. **Supabase 常量**放 `configs/supabase.env`（入库）；**`.env`** 放应用配置；**service_role** 仅 `.env.local`；推送前 `make check-secrets`

## 相关文档

- [启动指南](./docs/startup-guide.md)
- [Supabase 集成说明](./docs/supabase-integration.md)
- Flutter [AGENTS.md](../../my_ai_project/AGENTS.md)
