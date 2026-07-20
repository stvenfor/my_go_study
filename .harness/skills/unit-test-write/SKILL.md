# Skill — unit-test-write

## When

实现改动了 usecase / repository / 中间件逻辑；或 Brief 验证命令指向的测缺失。

## Do

1. 在白名单内补/改 Go 测试（与现有 `*_test.go` 风格一致）
2. 断言对齐已声明行为或 Degraded note
3. 把命令写进 Slice「验证命令」（若尚未写）

## Don't

- 为过测而削弱生产断言
- 跨包拉私有测试 helper（除非已在 `pkg` / 共享 testutil）
