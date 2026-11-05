# 学习记录 0001 - Casbin 与 Viper 概览

## 学习时间
2026-11-03

## 核心结论
- **Viper**：Go 应用配置管理库，支持 YAML/JSON/TOML/HCL/Env 等多种格式，支持环境变量覆盖、配置热重载、默认值等。当前项目 `common/config/config.go` 已在使用，通过 `viper.New()` 加载 `config.yaml` + `config.<env>.yaml`。
- **Casbin**：Go 权限访问控制库，支持 ACL、RBAC、ABAC 等多种模型。当前项目 `core/authz/` 新增引入，基于 `gorm-adapter` 将策略持久化到 PostgreSQL 的 `casbin_rule` 表。
- **关系**：互补，不冲突。Viper 在启动期读取配置；Casbin 在运行期做授权决策。

## 待验证
- `core/authz` 包在 `app` 模块中的实际集成（DI 容器注入 Enforcer 到 usecase/handler）
- 策略种子数据的初始化方式（启动时灌入默认策略 vs 运行时动态管理）

## 下一步
- 在 `app/cmd/api/main.go` 中将 `authz.Enforcer` 接入 DI
- 在 `app/internal/access/service.go` 的 `can`/`require` 中可选替换为 Casbin Enforce