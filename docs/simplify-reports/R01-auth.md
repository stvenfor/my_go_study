# R01 — Auth / Session（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R01 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Brief | [plans/slices/simplify-R01-auth.md](../../plans/slices/simplify-R01-auth.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. 优化前问题（Before）

### 1.1 结构 / 对应
- Go Auth/Session 与 Flutter `features/auth` 配对完整。
- 死代码：`SupabaseAuth` 中间件无调用方；`LogoutRequest` DTO 未使用。

### 1.2 Quality
- Flutter `AuthHttpConfig.ensureInitialized` 每次 `reinitialize`，会冲掉壳层 SessionGuard / Refresh interceptor（`HomeHttpConfig` 已是已初始化则 return）。
- `UserAuthApi` 7 个端点重复 try/catch + `_mapFailure`。
- Go `RenewOnRefresh` 三处几乎相同的 `DeviceSession`+`Save`。

### 1.3 Perf
- Auth 侧 Dio 重复重建（同上）。

### 1.4 Reuse
- 应对齐 `HomeHttpConfig` 的「已初始化则跳过」。

### 1.5 风险与不做（已标 SKIP）
- `handleSessionError` 与 middleware abort 默认码不同（500 vs 401），不合并。
- Session Validate/Renew 平台规则、logout 文案启发式、OAuth usecase 抽取 — 不动。

## 2. 优化目标（Goals）

- G1：`AuthHttpConfig` 已初始化则 no-op。
- G2：`UserAuthApi` 内联请求守卫，语义不变。
- G3：`RenewOnRefresh` Save 去重为私有 helper。
- G4：删除死代码 `SupabaseAuth`、`LogoutRequest`。
- 非目标：契约、错误码、session 规则。

## 3. 优化后结果（After）

### 3.1 已落地（对照 Goals）

| Goal | 改动摘要 | 行为是否不变 | 验证 |
|------|----------|--------------|------|
| G1 | `features/auth/.../auth_http_config.dart` | 是 | dart analyze features/auth（无 error） |
| G2 | `features/auth/.../user_auth_api.dart` `_guarded` | 是 | 同上 |
| G3 | `device_session_usecase.go` `saveDeviceSession` | 是 | `go test ./internal/usecase` ok |
| G4 | `supabase_auth.go` 删死中间件；`user_request.go` 删 LogoutRequest | 是 | `go test middleware` + `go build ./cmd/api` ok |

### 3.2 Diff 量级
- 两端共约 5 文件；删除死代码 + 内联重复；无新抽象/依赖。

### 3.3 Ponytail 跳过项（Skipped）

| 项 | 卡在第几阶 | 原因 |
|----|------------|------|
| handleSessionError↔middleware | 1 YAGNI / 行为风险 | HTTP 默认码不同 |
| BackendUser 冗余字段 | skip | 契约风险 |
| IssueOnLogin vs Renew 平台规则统一 | skip | 需产品确认 |
| deviceId helper / attachDeviceSession / 注释横幅 | 1 YAGNI | 本轮收益不足 |

### 3.4 残余与下一刀
- 无阻塞项；下一轮 R02 Profile。
