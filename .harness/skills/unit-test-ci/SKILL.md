# Skill — unit-test-ci

## When

实现或补测结束后、宣称可审查前。

## Do

```bash
# 若 harness 已落地：
make agent-post
# 或显式：
make agent-post SLICE=plans/slices/<id>.md

# 未落地 harness 时，至少跑 Brief「验证命令」，例如：
make test
# 或聚焦：
go test ./internal/usecase/...
```

确认 harness-report 中 `post.ok === true`，或 Brief 列出的命令全部 exit 0。

## Don't

- 默认跳过验证
- 验证红仍勾验收

## Also

专题脚本须与 Brief 一致，例如：`make test-transactions`、`make test-realtime`、`make check-secrets`。
