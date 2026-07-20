# Rule — auth token hygiene

本仓两套认证 **勿混用**：

| 中间件 | 用途 | Token |
|--------|------|-------|
| `SupabaseSessionAuth` | 业务 API（transactions / realtime / profile…） | Supabase access token + Redis session |
| `Auth`（JWT） | 遗留 `/api/v1/user/list` 等 | Go 自建 JWT |

硬约束：

- Flutter / 业务路径走 Supabase token + `X-Session-ID` / `X-Device-ID`
- `SUPABASE_SERVICE_ROLE_KEY` 仅 `.env.local`；推送前 `make check-secrets`
- PostgREST 业务请求 `WithUserToken`，禁止 Admin 绕过 RLS

权威：仓库根 `AGENTS.md` · `docs/auth-beginner-walkthrough.md`
