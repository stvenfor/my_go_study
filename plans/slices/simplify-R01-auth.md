# Slice — simplify-R01-auth

## Slice Brief
- SOURCE_MODULE: Go Auth/Session
- TARGET_MODULE: Flutter features/auth
- 本轮 ONLY: 行为不变的删除/内联/复用（Ponytail）
- 不做: 改 session 规则、错误码语义、handleSessionError↔middleware 合并、契约字段
- 验收: 机跑绿 + 报告 §3；预批连续执行
- 文件白名单:
  - my_go_study: internal/delivery/http/handler/user*, middleware/local_session_auth.go, supabase_session_auth.go, supabase_auth.go, device_session_error.go, supabase_session_auth_test.go, usecase/device_session_usecase.go, dto/request/user_request.go
  - my_ai_project: features/auth/lib/api/auth_http_config.dart, user_auth_api.dart, features/auth/lib/session/, commons/network/lib/http/auth_header_provider.dart（仅若必需）
- 文件黑名单:
  - UI pages、wechat/huawei usecase 逻辑、_mapFailure 文案表
- 验证命令:
  - cd my_go_study && go test ./internal/delivery/http/middleware/ ./internal/usecase/ -count=1
  - cd my_go_study && go build ./cmd/api
  - cd my_ai_project && dart analyze features/auth
- 证据: docs/simplify-reports/R01-auth.md
- AcceptMode: Partial（机跑；iOS 烟测账号 13400000000/123456 在关键轮后）

## Context Card — simplify-R01-auth
- 已完成: （执行后填）
- 未做/Deferred: handleSessionError 合并等 SKIP
- harness: post ok?
- 下一 Slice 建议: R02 profile
