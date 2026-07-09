# 认证（Auth）初学者逐行导读

> Flutter 邮箱密码登录/注册 → Go BFF → Supabase Auth，返回 **access_token** 供后续业务 API 使用。

协议与配置见 [supabase-integration.md](./supabase-integration.md)。

---

## 0. 三句话理解认证链路

```
1. Flutter  POST /api/v1/user/login  { username: 邮箱, password }
2. Go       代理 Supabase SignInWithEmailPassword
3. Flutter  保存 data.token → 之后每个业务请求 Header: Authorization: Bearer <token>
```

**为什么不直连 Supabase SDK？**

- 统一 BFF：密钥、错误文案、日志都在 Go 控制
- Flutter 只认 Go 的 `ResultModel` 信封，降低客户端复杂度

---

## 1. 推荐阅读顺序

### Go（my_go_study）

| 顺序 | 文件 | 内容 |
|------|------|------|
| 1 | `pkg/auth/supabase.go` | 如何校验 token（中间件用） |
| 2 | `internal/delivery/http/middleware/supabase_auth.go` | Gin 中间件注入 userID |
| 3 | `internal/usecase/supabase_auth_usecase.go` | 注册/登录业务 |
| 4 | `internal/delivery/http/handler/user_handler.go` | HTTP 入口 + 错误映射 |

### Flutter（my_ai_project）

| 顺序 | 文件 | 内容 |
|------|------|------|
| 1 | `packages/features/auth/lib/api/user_auth_api.dart` | HTTP 调用 + 错误分类 |
| 2 | `packages/features/auth/lib/session/backend_auth_service.dart` | 登录态持久化 |
| 3 | `packages/commons/network/lib/http/auth_header_provider.dart` | 自动带 Bearer token |

---

## 2. 登录请求/响应示例

**请求**

```http
POST /api/v1/user/login
Content-Type: application/json

{"username":"you@example.com","password":"123456"}
```

> `username` 必须是**完整邮箱**（Go 会检查含 `@`）。

**成功响应（ResultModel 信封）**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJ...",
    "user": {
      "id": "uuid",
      "username": "display_name",
      "email": "you@example.com"
    }
  },
  "timestamp": 1749456789
}
```

**常见错误码**

| code | 含义 | Flutter 映射 |
|------|------|--------------|
| 10002 | 密码错误 | `InvalidCredentialsFailure` |
| 10003 | 账号未注册 | `AccountNotRegisteredFailure` |
| 10001 | 参数/邮箱验证 | `WeakPasswordFailure` 等 |
| 50000 | 服务端异常 | `UnknownAuthFailure` |

---

## 3. 分层职责（Clean Architecture）

```
UserHandler          ← 解析 JSON、选 HTTP 状态码
    ↓
SupabaseAuthUsecase  ← 调 Supabase、映射业务错误
    ↓
pkg/supabase.Client  ← gotrue-go SDK
```

**为什么错误在 Usecase 用 `errors.Is` 而不是直接写 HTTP 码？**

- Usecase 可被 CLI/测试复用，不应知道 HTTP
- Handler 的 `handleUsecaseError` 统一映射，改文案只改一处

---

## 4. Token 校验（业务 API 怎么用登录结果）

```go
// 路由组挂载中间件
v1.Use(middleware.SupabaseAuth(cfg.Supabase))

// 控制器里取用户
user, token, ok := supabaseAuthContext(c)
// user.ID 就是 Supabase UUID，用于 transactions user_id 过滤
```

中间件调用 `GET {SUPABASE_URL}/auth/v1/user`，token 无效则 **401**。

---

## 5. service_role 的作用（可选）

`.env.local` 中的 `SUPABASE_SERVICE_ROLE_KEY` **不入库**。

用途：登录失败时区分「未注册」vs「密码错误」（`refineInvalidCredentials` 查 Admin API）。

未配置时，两种失败都显示「密码错误」——仍安全，只是提示不够精确。

---

## 6. Flutter 登录后发生了什么

```dart
// 1. UserAuthApi 解析 ResultModel
final result = await _api.login(username: email, password: password);

// 2. BackendAuthService 写入本地
await _userService.setUser(User(..., token: result.token));

// 3. 后续 HttpManager 自动注入 Authorization（AuthHeaderProvider）
```

Realtime / transactions 都依赖这个 token，**过期需重新登录**。

---

## 7. 本地调试

```bash
cd my_go_study && make run

curl -X POST http://127.0.0.1:8080/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"you@example.com","password":"yourpass"}'
```

Flutter：`.env` 中 `USE_MOCK_AUTH=false`，Hot Restart 后走真实登录。

---

## 8. 相关文档

- [Supabase 集成说明](./supabase-integration.md)
- [Transactions 初学者导读](./transactions-beginner-walkthrough.md)
- Flutter [BACKEND_INTEGRATION.md](../../../my_ai_project/docs/BACKEND_INTEGRATION.md)
