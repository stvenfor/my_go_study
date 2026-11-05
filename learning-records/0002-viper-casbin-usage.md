# Viper 与 Casbin 在 my_go_study 中的使用与注意事项

> 配套笔记：`learning-records/0001-casbin-viper-intro.md`（概念概览）。本文讲**实际用法 + 坑**。
> 更新日期：2026-11-03 · Git 为真相源，本文可经 lark-cli 同步到飞书知识库。

---

## 0. 一句话定位

- **Viper**：启动期**配置管理**（读 YAML / env，解析成结构体）。一次性加载，不热重载。
- **Casbin**：运行期**授权决策**（RBAC：谁对什么资源能做什么）。
- 两者**互补不冲突**：Viper 在启动期把 `authz.provider` 等开关读进来；Casbin 在请求期用这些配置下的策略做 `Enforce`。

> ⚠️ 仓库里存在**两套 Viper 实现**：
> - ✅ 活用的：`common/config/config.go`（`config.Load` / `ResolveConfigDir`）。
> - ❌ 死模块：`thunder/config/viper.go`（`thunder` 模块零引用，不被任何代码 import）。那里有 `InitConfig()`/`Init()` 两个初始化函数和 `WatchConfig` 热重载——**不要参考它**，仓库真正跑的是前者，且**没有热重载**。

---

## 1. Viper 实战

### 1.1 接入口

唯一入口是 `common/config/config.go:712` 的 `Load(configPath, env) (*Config, error)`。

调用链（`app/cmd/api/main.go:83`）：

```go
cfg, err := config.Load(config.ResolveConfigDir(), env)
```

- `ResolveConfigDir()`（`config.go:968`）：从当前工作目录**向上递归**找 `configs/` 目录。解决了「IDE 从子目录启动找不到配置」的问题。
- `env` 来自 `APP_ENV`（dev / lan / prod…），**与 gin 的 `server.mode`（debug/release）是两回事**，别混。

### 1.2 加载流程（按顺序）

1. `viper.New()`（**每个进程一个独立实例，不是全局单例**——比 `viper.ReadInConfig()` 全局态更安全）。
2. `AddConfigPath(configPath)` + `SetConfigName("config")` + `SetConfigType("yaml")` → 读 `configs/config.yaml`。
3. 若 `env != ""`：`SetConfigName("config."+env)` 再 `MergeInConfig()` 合并 `configs/config.<env>.yaml`（缺该文件时回退基础配置，不报错）。
4. `SetDefault("database.auto_migrate", false)` 等少量默认值。
5. `SetEnvKeyReplacer(strings.NewReplacer(".", "_"))` + `AutomaticEnv()`，再 `BindEnv` 约 70 个键（`server.port`↔`SERVER_PORT`、`database.host`↔`DATABASE_HOST`…）。
6. `Unmarshal(&cfg)`：按 **`mapstructure` tag** 填进 `Config` 结构体。
7. 另用独立 viper 实例读 `configs/supabase.env`、`.env`/`.env.local`（type=`env`），仅填空的字段；service_role 只允许来自 OS env 或 `.env.local`。
8. `validateSecrets` + 一堆 `if x <= 0 { x = default }` 兜底。

### 1.3 配置优先级（高 → 低）

```
OS 环境变量  >  .env / .env.local / supabase.env  >  config.<env>.yaml  >  config.yaml  >  SetDefault
```

注意 `AutomaticEnv()` + `BindEnv` 是**最高优先级**，所以容器/CI 注入 env 即可覆盖一切，不用改 YAML。

### 1.4 如何新增一个配置项（范例）

假设要给 SSE 加一个 `sse.max_connections`：

1. 在 `Config` 里加字段：`MaxConnections int \`mapstructure:"max_connections"\``（注意是 **snake_case**，对应 YAML 键）。
2. 在 `Load` 里加 `BindEnv("sse.max_connections", "SSE_MAX_CONNECTIONS")`。
3. 在 `applySSEDefaults` 里加兜底：`if cfg.SSE.MaxConnections <= 0 { cfg.SSE.MaxConnections = 100 }`。
4. YAML 里写：
   ```yaml
   sse:
     max_connections: 100
   ```
5. 重启进程生效（见下方注意事项 #1）。

### 1.5 注意事项

1. **无热重载**：`common/config` 只用 `Load` 在启动期读一次，**没有 `WatchConfig`**。改 YAML 后必须**重启进程**才生效。别被死模块 `thunder/config/viper.go` 里的 `WatchConfig` 误导。
2. **`mapstructure` tag 必填**：字段不写 tag 就读不到对应 YAML 键；键名是 snake_case，结构体是 CamelCase，靠 tag 桥接。
3. **bool 零值陷阱**：viper 的 bool 没配置时零值是 `false`，没法区分「没配」和「配了 false」。仓库用 `SetDefault("database.auto_migrate", false)` + `applySchemaPolicy` 强制在 prod/lan 关 DDL 来规避（`config.go:735`、`:916`）。自己加 bool 配置项时想清楚零值语义。
4. **`config.<env>.yaml` 可缺省**：prod/staging 常只有基础 `config.yaml`，`MergeInConfig` 报 `ConfigFileNotFoundError` 时会被吞掉并打日志回退（`config.go:725`）。别以为「配了 env 就一定有对应文件」。
5. **env key 分隔符**：`AutomaticEnv` 配合 `SetEnvKeyReplacer(".", "_")`，所以 YAML 的 `database.host` 对应环境变量 `DATABASE_HOST`（点变下划线，全大写）。BindEnv 里手写的也必须遵循这个命名。
6. **依赖升级要钉版本（硬规则 AGENTS.md #9）**：**绝不要盲目 `go mod tidy`**。实测 tidy 会把 viper `1.19 → 1.21`、zap `1.27 → 1.28`、gorm `1.25.12 → 1.31.2` 全拉到最新。要升级 viper 必须：先在 `common/go.mod` 显式写 pinned 版本，再 tidy，最后全量 `make test`。
7. **密钥不要进 YAML**：`jwt.secret` 为空直接报错（`config.go:839`）；`supabase.service_role_key` 只允许 OS env / `.env.local`，且 prod 有校验（`validateSecrets`）。提交前务必 `make check-secrets`。
8. **`ResolveConfigDir` 依赖 CWD**：它从「当前目录向上找 `configs/`」。如果你 `cd` 到别的目录再 `go run`，可能找不到配置——集成/容器里要确保工作目录正确，或用绝对路径。
9. **结构体默认值写法的两种流派**：本仓库大量用「`GetXxx()` 方法在 nil 时返回默认值」（见 `thunder/config` 的范式，虽是死模块，但思路可借鉴）；`common/config` 用的是「`Load` 末尾 `if <=0` 兜底」。新增配置项统一用后者的风格，保持与 `common/config` 一致。
10. **`Unmarshal` 是浅覆盖**：嵌套结构（`queue.rabbitmq.url`）靠点分 key 映射，YAML 缩进错了会静默读不到，编译不报错。改 YAML 后跑一下启动确认没 panic。

---

## 2. Casbin 实战

### 2.1 本仓库封装

`core/authz/casbin.go` 封装了 `casbin.Enforcer`：

- `NewEnforcer(db *gorm.DB) (*Enforcer, error)`（`casbin.go:36`）：用 `gorm-adapter` 的 `NewAdapterByDB(db)` 接现有 Postgres，**策略持久化到 `casbin_rule` 表**（适配器自动建表）；加载同目录的 `model.conf`（路径用 `runtime.Caller` 取绝对路径，避免依赖 CWD）。
- 暴露的 Go 风格 API：`Enforce(sub,obj,act)`、`AddPolicy`、`RemovePolicy`、`GetPolicy`、`AddRoleForUser`、`DeleteRoleForUser`、`GetRolesForUser`、`GetUsersForRole`、`SavePolicy`。`EnforceMatch` 只是 `Enforce` 的别名，**没有**额外的通配匹配能力。

### 2.2 模型与匹配语义（关键）

`core/authz/model.conf` 的 matcher 是：

```
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

要点：

- **精确相等，无通配符**：`r.obj == p.obj` 是字符串全等。Casbin 默认**不做** `keyMatch2` 之类的路径通配——`/api/v1/transactions/123` **不会**匹配策略里的 `/api/v1/transactions/*`。要做路径/前缀匹配，必须在 `model.conf` 的 matcher 里显式用 `keyMatch2(r.obj, p.obj)`。
- **`act` 被集成层硬编码为 `"access"`**：见 `access/service.go:288` 调用 `s.enforcer.Enforce(actorID, permission, "access")`。所以本仓库模型里 `act` 这一列形同固定值，真正区分权限的是 `obj`（即 `permission` 字符串）。
- **门店维度用「合成 obj」绕过**：`access/service.go:296` 把 `permission + ":store:" + storeID` 拼成新 obj 再 `Enforce` 一次（如 `member.write:store:5`）。这是一种**约定式 hack**，不是 Casbin 原生层级资源——加门店级权限时务必按这个约定写策略。

### 2.3 启动接线（已是可选 + 优雅降级）

`app/cmd/api/main.go:195-202`：

```go
casbinEnforcer, casbinErr := authz.NewEnforcer(db)
if casbinErr != nil {
    log.Warn("Casbin Enforcer 构造失败，权限判定回退原有 SQL 逻辑", zap.Error(casbinErr))
} else {
    accessSvc.SetEnforcer(casbinEnforcer)   // 注入；nil 时 access 走原 SQL
    log.Info("Casbin Enforcer 已就绪")
}
```

`app/internal/access/service.go` 的 `can`/`require`：**先问 Casbin，命中即放行；未命中或 enforcer 为 nil 再回退原有 SQL 权限码检查**（`service.go:287-303`）。所以 Casbin 目前是**渐进式接入**，旧 SQL RBAC 仍保底。

### 2.4 API 速查

```go
enf, _ := authz.NewEnforcer(db)

// 决策
ok, _ := enf.Enforce("user-123", "member.write", "access")          // 角色继承已内置（g 关系）
ok, _  = enf.Enforce("user-123", "member.write:store:5", "access")  // 门店维度

// 角色 / 用户
enf.AddRoleForUser("user-123", "store_admin")
enf.GetRolesForUser("user-123")
enf.DeleteRoleForUser("user-123", "store_admin")

// 策略（写后即落 casbin_rule 表）
enf.AddPolicy("store_admin", "member.write", "access")
enf.RemovePolicy("store_admin", "member.write", "access")
enf.GetPolicy()   // 全部 [sub, obj, act]
enf.SavePolicy()  // 批量改写落库（单条 AddPolicy 适配器已即时落库，SavePolicy 用于批量后整体刷）
```

### 2.5 如何新增一条授权

1. 决定 `sub` 用角色还是用户；`obj` 用 `permission` 字符串（带门店就 `+":store:"+id`）；`act` 固定 `"access"`。
2. 启动时或运行时 `AddPolicy` / `AddRoleForUser`（落库即生效，无需重启——前提是**同一进程内存已 LoadPolicy**，见下条）。
3. 业务侧调用 `accessSvc.can(ctx, actorID, "member.write", &storeID)`，由 `can` 统一走 Casbin 或回退 SQL。

### 2.6 注意事项

1. **策略在内存，跨实例不自动同步（最重要）**：`LoadPolicy()` 在 `NewEnforcer` 时把 `casbin_rule` 全量载入**本进程内存**，`Enforce` 是内存匹配，快。但另一个 API 副本改了 DB 里的策略，**本副本不会自动感知**——多实例（lan / prod 多副本）下会出现「策略改了但有的实例还没生效」。解决：改完策略后触发各实例 `LoadPolicy()` 重载，或用单实例 / 接受最终一致。
2. **`Enforce` 默认不认通配符**：如上，路径/前缀/层级资源不会自动匹配。需要通配就在 `model.conf` 引入 `keyMatch2`/`keyMatch3` 等内置函数改写 matcher，并重新 `LoadPolicy` +（若改 model）重部署。
3. **`SavePolicy` vs 单条写**：`gorm-adapter` 的 `AddPolicy`/`AddRoleForUser` **每次都即时写库**；`SavePolicy()` 是把内存整份刷回库，适合批量构造后一次性提交。别把二者混用导致重复写入。
4. **`model.conf` 是代码，不是数据**：模型（谁继承谁、怎么匹配）改了必须**重部署 + 重新 LoadPolicy**；而策略（`casbin_rule` 表的行）是数据，运行时增删即生效。两者别搞混——改 matcher 不上线等于没改。
5. **两套 RBAC 并存，谨防漂移**：`app/internal/access` 的自建 SQL RBAC 与 `core/authz` 的 Casbin 同时生效。`can` 是「Casbin 优先、SQL 兜底」，意味着**同一权限可能两条路径都算数**。新业务优先写 Casbin；老 SQL 权限码别再扩，避免两套口径长期分裂。
6. **`casbin_rule` 表自动建表与 DDL 闸门冲突待验证**：`gorm-adapter` 在 `NewAdapterByDB` 时通常会对 `CasbinRule` 模型 `AutoMigrate`。但本仓库在 prod/lan 强制 `database.auto_migrate=false`（`applySchemaPolicy`，`config.go:922`）。需确认适配器建表在 DDL 关闭环境下**不会失败或静默被拒**；若会，应在发布流程里手动管这张表，或验证适配器用的不是受控的 GORM 连接。
7. **构造失败被吞成 Warn**：`main.go:198` 把 Casbin 构造失败降级为 Warn 并回退 SQL——上线后若 `casbin_rule` 表缺失/迁移未执行，你只会看到一条 Warn 日志，**权限悄悄全走旧 SQL**。排障时先 grep 这条 Warn。
8. **`Enforce` 返回 error 要处理**：集成层把 `Enforce` 的 error 包成 `fmt.Errorf("casbin enforce 失败: %w")` 并返回 403 路径；别忽略这个 error，否则策略引擎异常会被误判成「无权限」。

---

## 3. 两者关系与落地建议

- **时间轴**：Viper 在启动期把 `queue.*` / `auth.provider` / `cs_agent.*` 等读进内存；Casbin 在运行期用 `authz.Enforcer` 做决策。前者是「静态接线」，后者是「动态判断」。
- **新增一个「可配置开关 + 受控权限」功能的标准刀路**：
  1. Viper 侧：在 `Config` 加字段 + `BindEnv` + 默认值（`common/config/config.go`）。
  2. Casbin 侧：在 `model.conf` 确定 matcher；运行时 `AddPolicy`/`AddRoleForUser` 灌策略。
  3. 业务侧：统一经 `accessSvc.can` / `require` 判定，享受 Casbin 优先 + SQL 兜底。
- **不要**：把密钥写进 Viper 的 YAML；指望 Casbin 自动通配路径；以为改 YAML 不用重启；在多实例下改策略不重载。

---

## 4. 参考文件

| 内容 | 路径 |
|------|------|
| Viper 加载入口（活用） | `common/config/config.go`（`Load` @712、`ResolveConfigDir` @968） |
| Viper 死模块（别参考） | `thunder/config/viper.go` |
| Casbin 封装 | `core/authz/casbin.go`（`NewEnforcer` @36） |
| Casbin 模型（精确匹配） | `core/authz/model.conf` |
| Casbin 启动接线（可选+降级） | `app/cmd/api/main.go:195` |
| Casbin 业务接入（can/require） | `app/internal/access/service.go:276` |
| 概念概览 | `learning-records/0001-casbin-viper-intro.md` |
