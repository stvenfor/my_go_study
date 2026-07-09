# 认证（Auth）初学者逐行导读

> Flutter 邮箱密码登录/注册 → Go BFF → Supabase Auth，返回 **access_token** 供后续业务 API 使用。

协议与配置见 [supabase-integration.md](./supabase-integration.md)。

---

## 0. 三句话理解认证链路

```
1. Flutter  POST /api/v1/user/login  { username, password, device_id, platform }
2. Go       代理 Supabase SignInWithEmailPassword，Redis 写入唯一 mobile session
3. Flutter  保存 data.token + data.session_id → 业务请求带 Authorization + X-Session-ID + X-Device-ID
```

**单设备登录**：同一账号全局仅允许 1 个 mobile 会话（android/ios）。新设备登录会覆盖 Redis 中的 session；旧设备下次请求收到 **401「账号已在其他设备登录」**。

**账号白名单（不互踢）**：配置 `auth.session_whitelist_user_ids` 或 `session_whitelist_emails` 的内部/测试账号豁免单设备限制，Validate 直接放行，登录不写 Redis。生产可用环境变量 `AUTH_SESSION_WHITELIST_USER_IDS`（逗号分隔 UUID）。

**为什么不直连 Supabase SDK？**

- 统一 BFF：密钥、错误文案、日志都在 Go 控制
- Flutter 只认 Go 的 `ResultModel` 信封，降低客户端复杂度

---

## 1. 推荐阅读顺序

### Go（my_go_study）

| 顺序 | 文件 | 内容 |
|------|------|------|
| 1 | `pkg/auth/supabase.go` | 如何校验 token（中间件用） |
| 2 | `internal/delivery/http/middleware/supabase_session_auth.go` | JWT + Redis session 校验 |
| 3 | `internal/usecase/device_session_usecase.go` | 登录签发 / 校验 session |
| 4 | `internal/usecase/supabase_auth_usecase.go` | 注册/登录业务 |
| 5 | `internal/delivery/http/handler/user_handler.go` | HTTP 入口 + 错误映射 |

### Flutter（my_ai_project）

| 顺序 | 文件 | 内容 |
|------|------|------|
| 1 | `packages/features/auth/lib/api/user_auth_api.dart` | HTTP 调用 + 错误分类 |
| 2 | `packages/features/auth/lib/session/backend_auth_service.dart` | 登录态持久化 |
| 3 | `packages/features/auth/lib/session/device_auth_context.dart` | device_id / platform |
| 4 | `packages/features/auth/lib/session/session_guard.dart` | 401 被动踢下线 |
| 5 | `packages/commons/network/lib/http/auth_header_provider.dart` | Bearer + session 头 |

---

## 2. 登录请求/响应示例

**请求**

```http
POST /api/v1/user/login
Content-Type: application/json

{"username":"you@example.com","password":"123456","device_id":"ios-device-uuid","platform":"ios"}
```

> `username` 必须是**完整邮箱**；`platform` 仅 `android` / `ios`。

**成功响应（ResultModel 信封）**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJ...",
    "session_id": "uuid",
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

### 2.1 测试环境手机号 OTP（dev bypass）

仅在 `server.mode=debug` 且配置了 `auth.dev_test_phone` / `auth.dev_test_otp` 时生效（见 `configs/config.dev.yaml`）。

**发送验证码**

```http
POST /api/v1/user/phone/otp/send
Content-Type: application/json

{"phone":"13400000000"}
```

测试号直接返回成功（不发真实短信）。

**校验并登录**

```http
POST /api/v1/user/phone/otp/verify
Content-Type: application/json

{"phone":"13400000000","otp":"123456","device_id":"test-device","platform":"ios"}
```

成功响应与邮箱登录相同（`token` + `session_id` + `user`）。非测试号或 `server.mode=release` 返回「短信登录暂未开放」。

```bash
make test-phone-otp-login
```

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
// 路由组挂载中间件（Supabase JWT + Redis session）
sbAuth := middleware.SupabaseSessionAuth(cfg.Supabase, deviceSessionUC)
v1.Use(sbAuth)

// 控制器里取用户
user, token, ok := supabaseAuthContext(c)
// user.ID 就是 Supabase UUID，用于 transactions user_id 过滤
```

中间件先调用 `GET {SUPABASE_URL}/auth/v1/user` 校验 token，再比对 Redis 中 `X-Session-ID` + `X-Device-ID`。不匹配则 **401**（含「账号已在其他设备登录」）。

---

## 5. service_role 的作用（可选）

`.env.local` 中的 `SUPABASE_SERVICE_ROLE_KEY` **不入库**。

用途：登录失败时区分「未注册」vs「密码错误」（`refineInvalidCredentials` 查 Admin API）。

未配置时，两种失败都显示「密码错误」——仍安全，只是提示不够精确。

---

## 5.1 账号白名单（可选）

`configs/config.yaml` 或 `config.dev.yaml`：

```yaml
auth:
  session_whitelist_user_ids:
    - "00000000-0000-0000-0000-000000000001"
  session_whitelist_emails:
    - "internal@example.com"
```

白名单账号：登录仍返回 `session_id`，但不写 Redis；业务 API 跳过 session 校验，可多设备并行。

---

## 6. Flutter 登录后发生了什么

```dart
// 1. 读取 device_id / platform，UserAuthApi 解析 ResultModel
final device = await DeviceAuthContext.resolve();
final result = await _api.login(..., deviceId: device.deviceId, platform: device.platform);

// 2. BackendAuthService 写入本地（token + sessionId + deviceId）
await _userService.setUser(User(..., token: result.token, sessionId: result.sessionId));

// 3. HttpManager 自动注入 Authorization + X-Session-ID + X-Device-ID
```

旧设备下次 API 返回 401 时，`SessionGuardHook` 自动登出并跳转登录页。

---

## 7. 本地调试

```bash
cd my_go_study && make run

curl -X POST http://127.0.0.1:8080/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"you@example.com","password":"yourpass","device_id":"test-device","platform":"ios"}'

# 单设备联调（需 make run + Redis）
make test-single-device-login

# 测试手机号 OTP（13400000000 + 123456，需 service_role）
make test-phone-otp-login
```

Flutter：`.env` 中 `USE_MOCK_AUTH=false`，Hot Restart 后走真实登录。

---

## 8. 相关文档

- [Supabase 集成说明](./supabase-integration.md)
- [Transactions 初学者导读](./transactions-beginner-walkthrough.md)
- Flutter [BACKEND_INTEGRATION.md](../../../my_ai_project/docs/BACKEND_INTEGRATION.md)
