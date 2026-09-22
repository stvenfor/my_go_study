# Rule — module boundary（Clean Architecture）

- 依赖方向：`delivery`（handler/controller/middleware/router/ws）→ `usecase` → `domain` / `repository` 接口
- 实现落在 `repository/postgres|redis|supabase` 与 `pkg/*`
- **禁止** handler/controller 直连 DB / Redis / PostgREST 客户端细节（经 usecase / repo）
- **禁止** PostgREST 用 Admin / `service_role` 绕过 RLS；业务查询必须 `WithUserToken`
- transactions 等用户数据必须按会话 `user_id` 过滤 + RLS 迁移（身份取自 Session Auth，勿依赖客户端再传 user_id）

权威说明：仓库根 `AGENTS.md` · `docs/supabase-integration.md`  
机检（若有）：`make agent-check-boundary` / Brief「验证命令」
