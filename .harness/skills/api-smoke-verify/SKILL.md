# Skill — api-smoke-verify

## When

Brief 要求联调 / smoke；**非**每个 Slice 必跑。

## Do

按发布范围执行（人确认环境已起）：

| 检查 | 命令 / 路径 |
|------|-------------|
| 健康 | `curl http://127.0.0.1:8080/health` |
| 单测/专题 | `make test` · `make test-transactions` · `make test-realtime` |
| 单设备登录 | `make test-single-device-login` |
| 手机 OTP | `make test-phone-otp-login` |
| 密钥 | `make check-secrets` |

记录：入口 → 关键路径 → 错误态；写入 acceptance-record。

## Don't

- 用 `/health` 替代 Brief 业务验证命令
- 在 Slice「不做」含联调时强行扩大范围
- 把 smoke 成功当成交互/客户端 Accept（除非 Brief 写明）

## Refs

- `/AGENTS.md` 本地联调表
- `.harness/skills/acceptance-record-writer/SKILL.md`
