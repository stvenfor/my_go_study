# 双仓编码流程 Playbook（Go BFF × Flutter）

> 从 `my_go_study` + `my_ai_project` 多轮实战提炼：需求怎么说、提示词怎么写、方案怎么切、落地踩过哪些坑、下一轮怎么开。  
> **真相源优先级：** Slice Brief > CONTEXT.md 术语 > AGENTS.md 边界 > 对话临场指令。  
> **配套：** [`.harness/README.md`](../.harness/README.md) · `~/.cursor/skills/migration-os-harness/`  
> **Flutter 侧入口：** [`my_ai_project/docs/coding-playbook-dual-repo.md`](../../my_ai_project/docs/coding-playbook-dual-repo.md)（镜像导航；全文以本文件为真相源）

---

## 0. 一句话心法

**先冻结合同，再写代码；机跑与人证分开验收；假完成不如 Partial + Deferred。**

双仓默认姿态：

| 仓 | 职责 | 验证底线 |
|----|------|----------|
| `my_go_study` | BFF / 表 / usecase / 权限口径 | `go test` 定向 + `go build ./cmd/api` |
| `my_ai_project` | UI / 路由 / 模块边界 / 联调壳 | `dart analyze` 触及包 + 真机人证 |

人闸门永远三道：**批 Brief → 人证/Accept → commit/push**。

---

## 1. 从会话提炼出的工作流（标准 8 步）

```text
① 需求收口（人）          ② 术语与 Avoid（CONTEXT）
        ↓                          ↓
③ Program / Epic 队首      ④ Slice Brief 落盘（ONLY/不做/白名单/验证/证据）
        ↓                          ↓
⑤ 人批 Brief               ⑥ executor：pre → 实现 → post → Context Card
        ↓                          ↓
⑦ reviewer / 人证 Partial|Full     ⑧ 人指令 commit/push；指针回 conductor
```

### ① 需求收口（人怎么说才省返工）

**有效需求句式**（实战中高命中）：

```text
目标能力：……
对标页面/模块：……（例：对标「新车成交」）
真假边界：假上传 / 假支付 / 只填 AppKey / 不接 OSS
范围全做 or 分期：C1…Cn；本期不做：……
前后端都要 or 只一端：……
遗留清单：密钥/厂商通道等可后置
```

**无效/易翻车句式：**

| 说法 | 为什么翻车 | 改成 |
|------|------------|------|
| 「美化一下 / 优化体验」 | Agent 自创视觉语言 | 「对标 XXX 页骨架：顶栏四格+吸顶 Tab+卡片行+FAB」 |
| 「都做了」无白名单 | 偷改首页宫格、待办卡规则 | 显式 ONLY + 黑名单 |
| 「参考某仓，全部引入」 | 拉进资源/SDK 包袱 | 「参考逻辑，不引入对方资源；AppKey 可空」 |
| 「修一下报错」不给栈 | 猜错层（UI vs Docker 镜像） | 贴完整断言/请求路径/`make lan-up` 状态 |

### ② 术语与 Avoid（CONTEXT）

凡跨模块易混概念，先写进 `CONTEXT.md`：

- **定义一句** + **Avoid 列表**（禁止同义词偷换）
- 典型已冻结对：门店职务 ≠ 权限角色；购车意向档 ≠ 跟进级别；店管可见性用 `role.assign_store` 不用职务数字

Agent 写代码前应对齐术语；否则会出现「用职务当店管」「客户端自映射 ABEH」这类硬伤。

### ③④ Program → Epic → Slice

| 层 | 产出 | 何时改 |
|----|------|--------|
| Program | 里程碑、**不做**、队首 Epic | 每周 / 扩 scope 人批后 |
| Epic | 差距表 → Slice backlog | 开模块前先 Audit |
| Slice | **唯一执行单位** | 每一 Agent 会话一主题 |

Brief **必填字段**（缺一勿开干）：

- `本轮 ONLY` / `不做`
- `文件白名单` / `黑名单`
- `验证命令`（机跑）
- `证据` 路径 + Accept 模式 `Full | Partial`
- 实现类：`Context Card` 模板

### ⑤ 人批 Brief

未批不得派 executor 扩写业务。人批可一句：

```text
批 plans/slices/<id>.md；按 ONLY 执行；commit/push 等我指令。
```

### ⑥ 执行合同

```text
你是 executor。唯一合同：plans/slices/<id>.md
先核对 Brief 字段齐全（或 agent-pre）；只改白名单。
结束：Brief 验证命令全绿 → 填 Context Card → 写 acceptance-record（可 Partial）
禁止：勾 Full Accept、擅自 commit/push、扩大 ONLY
```

双仓 Slice 时在 Brief 白名单**同时列出**两端路径前缀；Go / Flutter 可并行仅当白名单不相交，且验收文档单写者。

### ⑦ 验收诚实枚举

| 决策 | 含义 |
|------|------|
| **Full Accept** | 机跑绿 + 人证清单全部有证据 |
| **Partial** | 机跑/代码落地绿；交互人证未做或未齐 → 未证项进 Deferred |
| **Deferred** | 明确不做/后置（深链、真 OSS、AppKey） |
| **Rework** | post 未绿 / 越界 / 对标失败 |
| **Blocked** | 需重切 Brief 或产品决策 |

**禁止：** `/health` 绿、`make run` 起得来、分析仅 info → 自称业务 Accept。

### ⑧ 收口与下一刀

- 更新 `.harness/changes/current.md`（Role / Next / Partial 证据路径）
- Context Card 交给下一会话即可，**不要**整段 transcript
- commit/push **仅人指令**；两端可同批，但 diff 范围仍受白名单约束

---

## 2. 提示词配方库（可复制）

### A. 规划（前后端方案，先不写码）

```text
结合当前双仓（my_go_study + my_ai_project）设计 <模块>：
1) CONTEXT 术语草案（含 Avoid）
2) 表 / API / 权限口径（对照已有成交/二手车/待办）
3) Flutter 对标页与模块边界
4) 切 Program/Epic/Slice backlog；标出假实现边界
先方案与 Brief 草稿，等我批再用 executor 落地。
```

### B. 执行 Slice（推荐默认）

```text
执行 plans/slices/<id>.md
先核对 ONLY/白名单/验证命令；只改白名单。
Go：定向 go test + go build ./cmd/api
Flutter：dart analyze 触及包；UI 对标 Brief「对标要点」表
结束写 docs/acceptance-records/YYYY-MM-DD-<id>.md（Partial 可，人证未做勿勾 Full）
commit/push 等我指令。
```

### C. 纯修 Bug / 回归（小刀）

```text
现象：……
证据：日志/截图/栈/请求路径
约束：不改需求语义；不扩大首页宫格/权限口径
先定位层（Flutter / BFF / Docker 镜像过旧 / 主题双真源），再最小修复。
```

### D. 密钥可空的三方接入

```text
参考 <路径> 的逻辑与深链，但不引入对方资源文件。
落地到「只填 AppKey/密钥即可启用」；其它链路写全。
产出：代码 + docs/<integration>.md 遗留清单（控制台/厂商通道/证书）。
```

### E. UI 对标（防「自创丑 UI」）

```text
对标页面：features/.../<reference>
必须同构：顶栏四格 / 吸顶 Tab / 卡片行 / 底 FAB / 空态
允许复制 widgets 到本模块目录，不强抽跨 feature 公共库
禁止裸 ListTile/ChoiceChip 堆叠冒充完成
Accept：人证并排对照；本轮 Partial 可
```

### F. 审查

```text
只 Review 不改代码。输入：Brief、Context Card、测试输出、diff、验收记录。
post/验证未绿 → Rework。输出 Approved|Rework|Blocked + P0/P1。
```

---

## 3. 修改方案怎么切（刀法）

### 3.1 先 Audit 再实现

Epic 差距表 ≥1 真实缺口后再开实现 Slice。反例：直接「整模块美化」→ 首页 9 宫格规则被改、待办深链被捎带。

### 3.2 竖切优先于横切

好刀：`C1 档案 CRUD` → `ui-parity` → `C2 流水` → `C4 店管口径`  
坏刀：一次改表 + 全 Flutter + 推送 + 主题。

### 3.3 权限与展示口径单独成 Slice

「谁能看见」与「好不好看」分开。店管用 `PermRoleAssignStore` / `role.assign_store`，**禁止**用门店职务数字。

### 3.4 假实现写进 ONLY

| 能力 | 允许的假 | 禁止冒充真 |
|------|----------|------------|
| 上传 | 本地选图 / 占位 URL | 真 OSS 配进本 Slice |
| 支付/登录/分享 | SDK 接线 + 空 AppKey | 声称已上架可用 |
| 视频 | DB 默认播放链 | 真转码上传 |
| 推送 | 登记/下发/深链协调 | 无 AppKey 却标 Fully Migrated |

### 3.5 双仓提交策略

- 同一产品切片可两端同日落地，**各自 commit message 说清 why**
- 未完成人证 → acceptance **Partial**，勿把「已 push」当成 Accept

---

## 4. 落地坑清单（会话实锤）

### 4.1 产品 / 范围

| 坑 | 表现 | 防线 |
|----|------|------|
| 范围漂移 | 「修跟进」顺手改首页最多 9 项、「更多」规则 | Brief 黑名单写死 `home_dashboard` 宫格逻辑 |
| UI 自创 | 跟进页「跟屎一样」相对成交页 | 强制对标表 + 复制成交 widgets 模式 |
| 无权限页丑 | 403 丢进通用加载失败 | 独立空态：文案「仅门店管理员」+ 返回 |
| 职务当权限 | 店管判断写错 | CONTEXT + ADR；对齐 `home_todo` |

### 4.2 联调 / 环境

| 坑 | 表现 | 防线 |
|----|------|------|
| Docker 镜像过旧 | 钱包页 404，代码仓里明明有路由 | 改 Go 后 **`make lan-up`**；先 curl 区分 404/401/200 |
| 主题双真源 | 仅 Nav/Tab 变暗，正文仍亮 | 单一 `themeMode` 真源；热**重启**非热重载 |
| 布局溢出 | `copyWith(fontSize)` 残留 `height` | 改字号必重设 `height`；宫格慎用固定行高+`.h` |
| iOS 打包断符号 | `kernel_snapshot_program` / 缺 RoutePath | 路由与 mock 符号与引用同步；分析触及包 |

### 4.3 契约 / 数据

| 坑 | 表现 | 防线 |
|----|------|------|
| ABEH↔高中低双写 | 客户端自映射漂移 | 写只收字母；意向档服务端派生 |
| `next_follow_up_at` 双写 | 档案与客户不一致 | usecase 单写入口 + 事务 |
| ILIKE 注入/性能 | 社区搜索不可用 | 统一 search API + escape + 必要时 pg_trgm |
| 播放器分叉 | 社区 vs 小视频两套 | 新能力先查是否已有 Kit；鸿蒙走 CPF/ohos 分支 |
| feature 互引 | 跟进硬依赖 settings 成交包 | 复制到本模块；路由/抽象服务跨模块 |

### 4.4 流程 / 验收

| 坑 | 表现 | 防线 |
|----|------|------|
| 假完成 | 机跑未绿就 Accept | `no-silent-accept`；Partial 明示 |
| 会话过载 | 一聊天串 JPush+搜索+跟进 | 一会话一 Slice 主题 |
| 并行互踩 | 两 Agent 改同一清单 | conductor 锁；验收单写者 |
| 密钥任务膨胀 | 引入参考仓资源 | 「逻辑可参考、资源不引入」写进 ONLY |

---

## 5. 角色与门禁（日常怎么开 Agent）

| 情境 | 开谁 | 人要做什么 |
|------|------|------------|
| 新模块 / 队首不清 | **planner** | 批 Brief |
| Brief 已批 | **executor** | 等人证后再 Accept |
| post 绿待审 | **reviewer** 或人看记录 | Approved 后才 commit |
| Rework 仍在白名单 | **executor** | — |
| 越界 / 要扩 ONLY | **planner** → 人再批 | 禁止 executor 私扩 |
| 多开并行 | **conductor** 裁决白名单 | 锁验收文档 |

指针文件：`.harness/changes/current.md`（Active slice / Role / Next / Partial 证据）。

---

## 6. 验证矩阵（复制到 Brief）

### Go

```bash
go test ./internal/usecase/ -count=1 -run '<FeatureRegex>'
go test ./internal/repository/postgres/ -count=1 -run '<RepoRegex>'   # 若有
go build ./cmd/api/
# 联调若走 Docker：改码后 make lan-up；curl 区分 401 vs 404
```

### Flutter

```bash
cd features/<module> && dart analyze lib/<area>
# 主题/原生插件变更：热重启或整包重跑，不靠热重载
# UI 对标：与参考页并排真机人证
```

### 证据文件最低结构

```markdown
# Acceptance — <id>
Decision: Partial | Full
## Verification（粘贴命令与 exit）
## Checklist evidence（行 → 文件/测试名）
## Interaction script（人证步骤）
## Deferred
```

---

## 7. 推荐「最小完备」目录约定

```text
CONTEXT.md                          # 术语 + Avoid
plans/YYYY-MM-DD-*-program.md       # 里程碑与不做
plans/epics/<epic>.md               # 差距表 + backlog
plans/slices/<id>.md                # 执行合同 + Context Card
docs/acceptance-records/…           # Partial/Full 证据
docs/adr/NNNN-….md                  # 权限口径等难逆决策
.harness/changes/current.md         # 本轮指针
docs/coding-playbook-dual-repo.md   # 本文
```

Flutter 侧遵守其 `AGENTS.md` 四层边界（`lib` / `commons` / `components` / `features`），**禁止 feature 互引页面/VM**。

---

## 8. 一周节奏（可执行日历）

| 日 | 动作 |
|----|------|
| 立项日 | Program 不做清单 + Epic 差距表；人批队首 Brief |
| 实现日 | 一 Slice 一会话；机跑绿 → Partial 记录 |
| 联调日 | `lan-up` + 真机人证脚本；主题/插件要热重启 |
| 收口日 | Full 或明确 Partial；人指令 commit/push；更新 current.md |
| 回顾 | 把新坑补进本文 §4；过时句删掉（防沉积） |

---

## 9. 反模式速查（看到就停）

1. 未批 Brief 直接大改双仓  
2. 用职务 / 统计卡数字当权限  
3. 客户端与服务端各维护一套枚举映射  
4. Docker 未重建却调「后端没接口」  
5. 热重载验证主题 / 原生插件  
6. 无对标页的「先做个能用的 ListView」冒充 UI Done  
7. push 成功 = 验收完成  
8. 一句话需求同时含：新模块 + 三方 SDK + 改首页信息架构  

---

## 10. 附录：本阶段实战切片地图（示例）

| 切片 | 仓 | 结果姿态 |
|------|----|----------|
| new-car-follow C1 CRUD | Go | Partial → 机跑 |
| new-car-follow ui-parity | Flutter | Partial → 待并排人证 |
| new-car-follow C2 流水 | 双仓 | Partial |
| new-car-follow C4 店管 | 双仓 | Partial；ADR 0016 |
| community-search | 双仓 | Partial |
| cash-wallet | 双仓 | 代码路径 + lan 验证；注意镜像 |
| jpush / fluwx+tobias | 双仓 | 接线完备 + 遗留清单（AppKey） |

队首与指针以 `.harness/changes/current.md` 为准。

---

## Changelog

| Date | Note |
|------|------|
| 2026-09-23 | 初版：从跟进/成交/搜索/钱包/推送/主题/首页宫格等会话提炼 |
