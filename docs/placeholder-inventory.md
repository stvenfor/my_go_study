# 假实现清单（Placeholder Inventory）

> 来源：`parity-p0-5-fake-impl-inventory`（`plans/slices/parity-p0-5-fake-impl-inventory.md`）。
> 口径：状态枚举只有五态，见 [`.harness/rules/status-enums.md`](../.harness/rules/status-enums.md)
> `Migrated | Degraded | Placeholder | Deferred | Removed`；`Degraded/Placeholder/Deferred` 必须有 note。
> 铁律：**能编译 / `/health` 通 ≠ Migrated**；未登记却声称完成 = 静默消失（禁止）。

## 怎么读这张表

| 字段 | 含义 |
|------|------|
| 文件:行 | 当前 HEAD 的证据位置（改动后行号已复核） |
| status | 五态之一 |
| note | 为什么是这个态、现在到底做了什么、没做什么 |
| 影响面 | 谁会被静默影响（Flutter / 三方 / 数据 / 运维） |
| 解除条件 | 满足什么才能改态，归哪个 Slice |

「未真实实现却返回成功」的分支一律改显式失败（`501` + 业务码 `10005`），或降级为
可观测的 `Degraded`（`degraded_path_total{name=...}`，见 `GET /metrics`）。

## 状态速览

| # | 条目 | status | Slice |
|---|------|--------|-------|
| 1 | 华为 IAP 服务端回调 stub | Placeholder（已改 501） | 待排（IAP 专项，P0-2 明确不做） |
| 2 | SMS 发送 worker stub | Placeholder | 待排（短信接入） |
| 3 | 手机号 OTP 发送 no-op | Placeholder（lab-only） | 待排（短信接入） |
| 4 | dev 固定测试号 + 固定码 `123456` | Degraded（lab-only，prod 拒绝） | — |
| 5 | 短视频 mock 图/视频 URL | Placeholder（lab-only） | 待排（对象存储） |
| 6 | 社区 mock 图/视频 URL | Placeholder（lab-only） | 待排（对象存储） |
| 7 | LLM mock provider + 无 key 静默回退 | Degraded | P2（真 LLM 配置） |
| 8 | 客服 Agent 不调 LLM（模板 + 伪流式） | Degraded | P2（Eino 接真模型） |
| 9 | analytics 演示种子 | Placeholder（lab-only） | — |
| 10 | 各域启动种子（10 处） | Placeholder（lab-only） | P0-1（迁到显式 migrate） |
| 11 | 三方登录占位邮箱 `wx_/hw_/dev.test.local` | Degraded | P2（真实邮箱绑定） |
| 12 | JPush 未配置 → `skipped` 但 HTTP 200 | Degraded | P2（极光凭据接入） |
| 13 | `ConfirmBuyout` 不校验支付即标记已付 | Migrated（渠道单需已核销流水） | P0-2（已验收） |
| 14 | 短视频「超时自动过审」 | Removed（已换人工放行） | P0-4（已验收） |
| 15 | 机审 / 驳回态 / 批量审核 UI | Deferred | P2-2 |
| 16 | `EnqueueSendSMS` 生产者无调用方 | Deferred | 待排（短信接入） |
| 17 | 清单扫描自动化门禁（`TODO/stub/mock`） | Deferred | 待排 |
| 18 | `membership_order` 业务核销器（回调即发权益） | Migrated（h8 已验收） | `harden-h8-membership-settler` |
| 19 | 对账只做「渠道查单」方向，`local_missing` 未实现 | Deferred | 待排（渠道账单文件下载） |
| 20 | 商城 Apple/华为 IAP 渠道仍本地标记已付、无渠道流水 | Placeholder | 待排（IAP 专项） |
| 21 | `unifiedorder` 响应未验签（只信回调报文） | Degraded | 待排（P2 支付加固） |
| 22 | 跨用户推送（`push/send`、`realtime/push` 群发他人） | Deferred | 另立 Slice（需 `access.Service` 注入 controller → 改 `main.go`，与 h8 冲突） |
| 23 | 定时广播删去重键失败（`_, _ = Del`） | Degraded（升级为硬失败 + 指标） | harden-h5 |
| 24 | 离线推送失败静默（`realtime_push`） | Degraded（指标留痕） | harden-h5 |
| 25 | 加成员后拉店群失败静默（`access`） | Degraded（指标留痕） | harden-h5 |
| 26 | 切店后拉店群失败静默（`profile`） | Degraded（指标留痕） | harden-h5 |
| 27 | 门店群 `CreateGroup` 失败被 `_ = err` 吞掉 | Degraded（指标留痕） | harden-h5 |
| 28 | **金额三套口径并存**（整数分 / decimal-as-string / float64） | Degraded | h6 审计交付；**h6b 已落 `transactions` 本地路径**（`amount_fen` 整数分 + 双写）；used_car / auto_finance / mall 仍是各自口径 |
| 29 | `auth.provider=supabase` 时收支金额不迁移 → 双 provider 口径不一致 | Degraded | h6b 验收第 5 条留痕；解除需云端补列 |
| 30 | mall 后台改价 `normalizeMoney` 静默四舍五入到 2 位 | Degraded | 待排（金额加固，h6 §4.6） |
| 31 | 两种库形态的 `amount_fen` 形状不同（迁移=NOT NULL+CHECK，AutoMigrate=可空无约束） | Degraded | h6b 已知坑；读路径靠 `*int64` 回落旧列兜住，约束仍只在 `migrations/` |
| 32 | 亚分金额（>2dp）被拒后返回 **502** 而不是 400 | Degraded | `app/internal/transaction/handler.go` 在 h6b 白名单外；需一条错误映射改动 |

关联但**由其它 Slice 收口**的假语义（此处只登记指向，避免双份真相）：
支付 `prepay` 生成即丢 + 无 notify 回调（P0-2）、MQ 失败无限重投无 DLX（P0-6）、
AutoMigrate/EnsureSchema 出启动流程（P0-1）、compose/配置的弱默认密钥（P0-7）。

---

## 1. 华为 IAP 服务端回调 stub

- 文件:行：`app/internal/membership/handler.go:152-160`（路由 `app/internal/membership/router.go:48`）
- status：`Placeholder`
- note：parity-p0-5 前直接 `response.Success({"accepted":true})`；现返回 **HTTP 501 + 业务码 10005**，
  文案「华为 IAP 回调未接入：当前不会处理该通知，请勿按成功记账」。没有验签、没有解析 `status`、没有落流水。
- 影响面：华为侧此前会误认为投递成功而**不再重试**，订阅续期/退款事件静默丢失；改 501 后华为按非 2xx 重试，
  本地也不会留下「已受理」的假象。路由现仍挂在 SessionAuth 之后（华为服务器无会话），这条路由本身要重建。
- 解除条件：IAP 专项 Slice（P0-2 明确划到「不做」范围：只做微信/支付宝 App 支付与回调）。
  需要：验签 + 解析 `status` + 落 `wys_payment_records`（`ref_type=membership_order`）+ 回调路由移出 SessionAuth。

## 2. SMS 发送 worker stub

- 文件:行：`app/internal/delivery/worker/handler.go:169-182`（载荷定义 `core/queue/tasks.go:36-39`）
- status：`Placeholder`
- note：反序列化后**只打一条 Warn**（`sms task NOT IMPLEMENTED: provider not integrated, task acked and dropped`），
  随后 `return nil` 即 ACK 丢弃。刻意不返回 error —— 返回 error 会走当前的无限重投（见 P0-6）。
- 影响面：任何入队的短信任务都不会发出，且队列侧看不出失败；worker 进程未暴露 `/metrics`，
  所以本片只给它**日志**，不给指标（指标会误导，没人抓取）。
- 解除条件：接入真实短信服务商（凭据 + 模板 + 回执），或明确删除该 task type。

## 3. 手机号 OTP 发送 no-op

- 文件:行：`app/internal/usecase/phone_otp_usecase.go:31-42`（Supabase 分支）、
  `app/internal/usecase/local_auth_usecase.go:196-201`（local 分支，dev/lan 实际走这条）
- status：`Placeholder`
- note：`POST /api/v1/user/phone/otp/send` 命中测试号时**什么都不做直接返回成功**，固定码由配置给定（见条目 4）。
  非联调环境（`APP_ENV=prod|production`）由 `config.AuthConfig.DevBypassEnabled` 拦截，返回
  `ErrPhoneLoginNotAvailable`。代码里不发消息、不落库、不计数。
- 影响面：Flutter 侧「已发送验证码」提示是假的（仅测试号）；真实号码走不到这条分支。
- 解除条件：与条目 2 同批（真实短信通道），届时把 no-op 换成真发送 + 验证码有效期与频控。

## 4. dev 固定测试号 + 固定码 `123456`

- 文件:行：门限 `common/config/config.go:306-311`（`DevBypassEnabled` = `IsLabAppEnv` + 非空测试号 + 非空码），
  判定 `app/internal/usecase/local_auth_usecase.go:196-210`、`app/internal/usecase/phone_otp_usecase.go:44-57`；
  值来自 `configs/config.dev.yaml`（`config.lan.yaml` 刻意留空）
- status：`Degraded`
- note：这是 AGENTS.md 写明的**联调依赖**（`13400000000` + `123456`），不删除、只按环境守卫。
  正式环境三条件里 `IsLabAppEnv` 先为 false → 发送/校验都返回 `ErrPhoneLoginNotAvailable`。
  回归断言：`app/internal/usecase/phone_otp_usecase_test.go` 的 `TestPhoneOTP_ProductionRejectsDevFixedCode`
  同时覆盖 local 与 Supabase 两个 provider 的 prod 拒绝。
- 影响面：若误在正式环境配置 `AUTH_DEV_TEST_*`，任何人可用固定码登录 → 现由代码门挡住，配置门另见 P0-7。
- 解除条件：无（长期保留的联调能力）；加固方向 = P0-7 启动期校验「正式环境禁止出现测试号配置」。

## 5. 短视频 mock 图/视频 URL

- 文件:行：默认值表 `app/internal/short_video/service.go:37-50`，守卫 `:136-141`，填充 `:339-350`
- status：`Placeholder`
- note：发布时若不传 `video_url/cover_url`，按 `uuid` 首字节取内置公网样例（bee.mp4 / oceans.mp4 / picsum 封面），
  并且**时长固定 `0:15`、宽高比固定 1.25**。parity-p0-5 起：`demoMedia = config.IsLabAppEnv(APP_ENV)`，
  正式环境缺真实 URL → `ErrInvalid`（HTTP 400），不再伪造媒体。
- 影响面：联调期 Flutter 播放的是外部站点视频（断网即黑屏）；正式环境现在会明确报错而不是入库假链接。
- 解除条件：接入对象存储直传（STS/签名 URL）后，把默认填充整段删除（属独立 Slice，未在本片实现）。

## 6. 社区 mock 图/视频 URL

- 文件:行：默认值表 `app/internal/community/service.go:31-38`，守卫 `:348-350` + 判定 `:85-94`，填充 `:585-618`
- status：`Placeholder`
- note：`media_type=image` 且不带图 → 塞 3 张 picsum；`media_type=video` 且缺 URL/封面 → 塞样例 mp4 + picsum 封面。
  同条目 5，正式环境（`demoMedia=false`）改为拒绝（`ErrInvalid` → 400）。
  `normalizeMedia` 本身保持原行为，便于单测（`app/internal/community/service_test.go:7-22`）。
- 影响面：同上；另外社区搜索/热度会统计到这些假媒体。
- 解除条件：同条目 5（对象存储）。

## 7. LLM mock provider + 无 key 静默回退

- 文件:行：`app/internal/repository/llm/mock_stream_provider.go:34-46`（关键词模板回复 `:74-105`），
  回退装配 `app/cmd/api/main.go:569-594`（`buildSSEProvider`）
- status：`Degraded`
- note：`sse.provider=openai_compatible` 但没配 `SSE_OPENAI_API_KEY` → 回退 mock；未配 provider 时也是 mock
  （parity-p0-5 前这条**连日志都没有**）。现两条路径都打 `Warn(status=degraded)`，
  且每次 mock 出话 `degraded_path_total{name="llm_mock"}` +1。SSE 帧里的 `model=mock-v1` 也可判别。
- 影响面：`POST /api/v1/sse/completions` 返回的是内置关键词回复，看似正常但与模型无关；
  演示可用，不能当真实 AI 能力对客宣传。
- 解除条件：正式环境强制要求非空 API Key（P0-7 启动校验）或直接 501 未开通；能力本身只需配置、无需改码。

## 8. 客服 Agent 不调 LLM

- 文件:行：`app/internal/usecase/cs_agent_usecase.go:99-101`（每次组答案计数）、
  `:156-176`（`buildCsAnswer` 纯模板拼接），伪流式 `:104-127`
- status：`Degraded`
- note：只有 Eino Tool 检索交易这一段是真的（且需 ES），**回答由模板拼接后按 rune 慢速吐出**，全程不调用模型。
  parity-p0-5 起计数 `degraded_path_total{name="cs_agent_template"}`。`cs_agent.enabled=false` 时路由仍注册但
  直接失败（`ErrCsAgentDisabled`）。
- 影响面：`POST /api/v1/cs-agent/chat` 的 SSE 成功语义 ≠ 智能客服；多轮/意图理解不存在。
- 解除条件：把 `buildCsAnswer` 换成真实模型调用（P2 客服 Agent 深化），届时移除计数与本条。

## 9. analytics 演示种子

- 文件:行：`app/internal/usecase/analytics_usecase.go:39-53`（`EnsureSeedData`）+
  `:80+`（`buildAnalyticsSeedRecords`），调用点 `app/cmd/api/main.go:405-407`
- status：`Placeholder`
- note：表内没有 `chart-v1` 标记时，清空 `source_system=seed` 并批量写入演示记录；parity-p0-5 起只在
  `IsLabAppEnv(APP_ENV)` 为真时执行（正式环境启动日志打「跳过演示种子」）。
- 影响面：gRPC `AnalyticsService/List` 与 Flutter 首页「数据分析」图表展示的是假数据。
- 解除条件：数据分析改为真实业务来源（或明确标注 demo 环境专属）；正式环境已不会自动写入。

## 10. 各域启动种子（10 处）

- 文件:行：统一入口 `app/cmd/api/main.go:597-610`（`seedIfLab`），调用点
  `main.go:201,206,209,216,221,226,232,233,314,405`；实现分布：
  `short_video/repo_seed.go:35`、`home/service.go:341` + `home/repo_seed.go:18`、
  `deal_invoice/repo_postgres.go:145`、`used_car/repo_postgres.go:143`、
  `new_car_follow/repo_postgres.go:252`、`purchase_calculator/repo_postgres.go:51`、
  `mall/repo_seed.go:19`、`community/service.go:100` → `community/repo_postgres.go:371-434`（话题/动态种子）
- status：`Placeholder`
- note：全部收口到 `seedIfLab`：`APP_ENV=prod|production` 时**一条种子 SQL 都不执行**，日志显式
  「跳过演示种子（APP_ENV 非联调环境）」。`EnsureSchema`（建表）未在本片处理 —— 那是 P0-1 的迁移闸门。
- 影响面：此前正式部署会往真实库写 `13400000000` 名下待办/订单/商品/动态，污染统计且难清理。
- 解除条件：P0-1 把建表 + 种子统一迁到显式 `make migrate-up` / 一次性容器后，本条可改 `Removed`。

## 11. 三方登录占位邮箱

- 文件:行：`app/internal/usecase/local_auth_usecase.go:212`（`{digits}@dev.test.local`）、
  `:259`（`hw_{unionID}@huawei.local`）、`:332`（`wx_{openid}@wechat.local`）
- status：`Degraded`
- note：本地 Auth 需要唯一 `email` 做账号锚点，三方只给 openid/unionID，故合成这些**不可投递**域名。
  `:292` 允许把 `@dev.test.local` 补齐，不会覆盖真实邮箱。
- 影响面：这类账号无法走「忘记密码 / 邮箱验证码」；任何按邮箱域做统计或风控的逻辑都会把它们算成异常域。
- 解除条件：绑定真实邮箱（或把 email 列改为可空 + 用 provider+openid 做唯一键）。属 P2 账号体系。

## 12. JPush 未配置 → skipped 但 HTTP 200

- 文件:行：`app/internal/usecase/jpush_usecase.go:72-75`（client nil）、`:100-102`（凭据缺失），
  调试路由 `app/internal/delivery/http/controller/jpush_controller.go:66`
- status：`Degraded`
- note：返回 `SendResult{Skipped:true, Reason:"..."}` 且 **error 为 nil**，因此 HTTP 仍是 200。
  刻意不在本片改成 501：这是「凭据未配」的正常态，且 `core/jpush/` 在 P0-5 黑名单里。
  可见性靠响应体的 `skipped/reason` 字段与 worker 日志；未加指标（需改 `jpush_usecase.go`，超白名单）。
- 影响面：`POST /api/v1/push/send` 与 WS 离线兜底推送在未配凭据时是 no-op；调用方若不看 `skipped` 会以为已送达。
- 解除条件：配置极光 AppKey/MasterSecret 后走真实下发（P2）；届时未配置分支应改 501 或启动期校验（P0-7）。

## 13. `ConfirmBuyout` 不校验支付即标记已付

- 文件:行：`app/internal/membership/service.go:283-295`（`ConfirmRefPaid` 前置校验 + 无 payment 域即拒绝）
- status：`Migrated`
- note：parity-p0-2 起，微信/支付宝渠道的买断必须先有**已核销流水**：
  `payment.ConfirmRefPaid` 先看本地 `wys_payment_records`（paid 即放行），仍 pending 就同步向渠道查一次单
  （回调常晚于 App 回跳）；查不到已付事实返回 `ErrPaymentNotConfirmed` → 不发权益。
  `payment` 未装配（`u.pay == nil`）同样拒绝，不再退化为「调用即已付」。
  残余：无 —— `MarkOrderPaid` 与 `ExtendEntitlement` 已并入同一 `txdb` 事务（见条目 18 的更新）。
- 影响面：客户端回跳不再是收款凭据；重复确认幂等（已 paid 直接回同一份权益）。
- 解除条件：不适用（本条已收口；事务化残余归条目 18）。

## 14. 短视频「超时自动过审」

- 文件:行：已删除的谓词 `dueForApprove`；现状 `app/internal/short_video/service.go:12-15`（头部语义说明）、
  人工放行 `Approve`（同文件）+ 路由 `POST /api/v1/short-videos/:id/review`
- status：`Removed`
- note：parity-p0-4 删除「等 N 秒即判通过」；待审内容对非作者永不可见，放行需 `role.assign_platform` 权限。
  演示模式 `review_mode=auto_allow_local_only` 仍需 `IsLabAppEnv` 才生效。
- 影响面：不再存在「无人审核却自动对外可见」的路径；历史遗留 `reviewing` 行需人工放行（SQL 见 P0-4 record）。
- 解除条件：不适用（该假实现已移除）。

## 15. 机审 / 驳回态 / 批量审核 UI

- 文件:行：不存在（`short_video` 域无 `rejected` 状态常量，无审核后台）
- status：`Deferred`
- note：P0-4 只做「默认拒绝 + 人工放行」，机审（图/文/视频风控）与驳回态属 P2-2；
  因此 `docs/placeholder-inventory.md` 不宣称短视频审核已 Migrated。
- 影响面：审核吞吐靠人工；被拒内容目前只能由作者自行删除。
- 解除条件：P2-2 接入内容安全 + 增加 `rejected` 状态与审核列表。

## 16. `EnqueueSendSMS` 生产者无调用方

- 文件:行：`core/queue/client.go:119-125`
- status：`Deferred`
- note：入队 API 已实现但除测试外**没有生产调用方**（`grep` 证据：仅 `core/queue/client.go` 自身与测试命中）。
  与条目 2 组合成「有人入队也发不出去，没人入队更发不出去」。
- 影响面：读代码者易误以为短信链路已通。
- 解除条件：接入短信通道时一并接上调用方；若确认长期不用，删除该端口方法并改 `Removed`。

## 17. 清单扫描自动化门禁

- 文件:行：不存在
- status：`Deferred`
- note：Brief 允许「成本高则 Deferred」。当前靠人工在改动时回写本清单 + `.harness` review 核对；
  未做 `TODO|FIXME|stub|mock` 自动扫描，因为仓内 `mock` 关键字大量存在于**测试**文件，需要设计排除规则与
  基线白名单，否则 CI 噪声大于价值。
- 影响面：新增假实现可能漏登记。
- 解除条件：约定注释标记（例如 `// placeholder:` 前缀）后，脚本只扫该标记即可低噪声落地。

## 18. `membership_order` 没有业务核销器 → **已收口（h8）**

- 文件:行：`app/internal/membership/service.go`（`SettleChannelPaid` + `NewService` 自注册）；
  原缺口 `app/internal/payment/settle.go:103-116`（`settled_no_settler` 分支，现仅作未知 ref_type 兜底）
- status：`Migrated`（`harden-h8-membership-settler` 已验收，证据见
  [`2026-11-05-harden-h8-membership-settler.md`](./acceptance-records/2026-11-05-harden-h8-membership-settler.md)）
- note：`membership.Service` 实现 `payment.Settler`，`NewService` 内 `pay.RegisterSettler(
  PaymentRefMembershipOrder, u)` 自注册（照 mall `WithPayment` 形态，**零 main.go 改动**）。
  回调核销 `membership_order` 即由 `SettleChannelPaid` 主动发权益，`settled_no_settler` 对该 ref_type 不再出现。
  写全部落在 payment 传入的 ctx 事务里（`MarkOrderPaid` / `ExtendEntitlement` 经 `txdb.DB`），体内零外部 HTTP/ES/MQ。
  幂等与「回调 vs `ConfirmBuyout` 双链并发」由 `MarkOrderPaid` 的 `WHERE status=pending + RowsAffected==0
  → ErrOrderPaid` 原子守卫保证，只加一个权益周期（真库用例 + 并发 x10 已证）。
- 影响面：回跳丢失 / 用户杀进程不再导致「付了钱没会员」；渠道收款与发货同事务，跨域「流水 paid、单未付、权益未发」
  的窗口（见 [`2026-11-04-membership-money-tx-followup.md`](./acceptance-records/2026-11-04-membership-money-tx-followup.md)
  §诚实口径 3）一并消失。
- 解除条件：已满足。剩余已知项（**均不在 h8 白名单，显式登记不静默丢弃**）：
  1. `payment/settle.go:105-108` 的注释仍把 `membership_order` 当「未登记」示例、且说「membership 仓储还不支持 txdb」
     —— 事实已变（注释陈旧），但该分支逻辑作为未知 ref_type 兜底仍正确；不在 h8 白名单，留给 payment 侧清理。
  2. F7「核销后通知用户」仍 `Deferred`，与本片无交集（h8 事务内零外部调用）。

## 19. 对账只覆盖「渠道查单」方向，`local_missing` 未实现

- 文件:行：`app/internal/payment/reconcile.go:55-92`（只产 `channel_missing` / `amount_mismatch` / `status_mismatch`）；
  常量 `model/entity/payment.go:PaymentDiffLocalMissing` 目前**无写入方**
- status：`Deferred`
- note：本片对账的实现形态是「本地 paid/pending 单逐笔向渠道查单」。
  「渠道账单里有、本地完全没这条流水」（例：商户号被别的应用共用、或本地落库前崩溃）需要先下载
  微信 `downloadbill` / 支付宝对账单文件并解析，属另一条数据链路，P0-2 明确不做。
- 影响面：`local_missing` 类差异现在**永远不会出现**，看告警的人容易误以为「无差异=两边一致」。
- 解除条件：接入渠道账单文件下载 + 解析 + 落 `wys_payment_reconcile_diff (kind='local_missing')`。

## 20. 商城 Apple / 华为 IAP 渠道仍是本地标记已付、不写渠道流水

- 文件:行：`app/internal/mall/service.go:505-517`（`PayOrder`：仅微信/支付宝走 `StartChannelPay`，其余渠道 `settleLocal`）
- status：`Placeholder`
- note：IAP 的收款事实由**客户端提供的 receipt/token** 证明，验签要打到 Apple/华为服务端，
  P0-2 的「不做」清单里排除了 IAP 改造，因此这两条渠道仍是「调用即已付」，且 `wys_payment_records` 里没有对应流水。
- 影响面：商城 IAP 订单不进对账；伪造 receipt 即可下单（会员侧 Apple 有 dev 放行，见条目 1 与 P0-5 record）。
- 解除条件：IAP 专项 Slice（验签 + 落流水 + 与 `membership` 共用 `ref_type` 口径）。

## 21. `unifiedorder` 响应未验签

- 文件:行：`app/internal/payment/service.go:242-259`（只看 `return_code`/`result_code`/`prepay_id`）
- status：`Degraded`
- note：P0-2 的硬门槛只约束**异步回调**报文（那是唯一能凭空造出「已付款」的入口）。
  预下单响应走 HTTPS 官方域名，未额外校验 `sign` 字段；若未来支持自建网关/代理（`WECHAT_PAY_ENDPOINT` 可配），
  响应伪造可拿到一个错误 `prepay_id`（结果是无法付款，不能凭空核销收款）。
- 影响面：不会造成资金误核销；最坏是下单链路被指到不可信端点后拿不到可用 `prepay_id`。
  `WECHAT_PAY_ENDPOINT` / `ALIPAY_GATEWAY` 只由运维配置注入，客户端不可控。
- 解除条件：对 unifiedorder 响应同样调用 `wechatSign` 校验（P2 支付加固）。

---

## 28. 金额三套口径并存（整数分 / decimal-as-string / float64）

- 文件:行（三种口径各自的代表，均为 2026-11-06 HEAD `e288553` 实测）：
  - **A 整数分 `int64`**：`model/entity/payment.go:53,82,84`、`model/entity/membership.go:48,63`、
    `model/entity/cash_wallet.go:30,40,41`、`model/entity/auto_finance.go:21,24`；
    DB `bigint` + `CHECK`（`migrations/20261104160000_payment_ledger.up.sql:8,21`、
    `migrations/20260923140000_membership_subscribe.up.sql:20`、`migrations/20260923095000_cash_wallet.up.sql:4`）
  - **B decimal-as-string**：`model/entity/mall.go:210,255,278,281,308` + 无 tag 的读模型 `:179,190,346`、
    `model/entity/membership.go:64`、`model/entity/cash_wallet.go:65,78`；DB `numeric(10,2)`
    （`migrations/20260921180000_mall_schema.up.sql:51,101,132,134,161,191`）
  - **C float64**：`model/entity/transaction.go:16,28,36`、`transaction_record.go:14`、
    `used_car_order.go:138,180`；`auto_finance.go:37-45,52,60-70` 的 float64 **只在对外的元读模型，不落库**
- status：`Partial`
- note：口径本身**分三套是历史事实**，不是 bug；bug 在于 **C 侧全链路没有任何一处做精度收敛** ——
  `grep -rnE 'Round|Trunc|decimal|ParseFloat' --include='*.go' app/internal/transaction/` → 0 命中，
  而 `transactions.amount` 的实际列型是**裸 `numeric`（无精度无标度）**
  （`migrations/20260923167500_align_gorm_only_schema.up.sql:40`，**不是** gorm tag 暗示的 float），
  所以 DB 能原样收下 `0.30000000000000004` 这类 float 累加残值，Go 读回即失真。
  A↔B 之间唯一的换算门 `entity.ParseYuanToFen`（`model/entity/cash_money.go:10-22`）
  是**拒绝式**（非整分即报错，不四舍五入），因此 **A/B 之间的边界已查证为安全**；
  `membership`↔`payment` **零换算**（`PriceFen` 直通 `AmountFen`）；`points` 与元之间**不存在任何汇率**
  （积分是 SKU 上的平行价，`MallSKUPaymentMode` 派生三态）。
  完整取证见 [`docs/money-unit-audit.md`](./money-unit-audit.md) §2/§3/§4（含 44 条字段清单与排除法残差 0 证明）。
- 影响面：Flutter 已**同时承受三套口径**（`double`：`transaction_model.dart:39`、`used_car_order_models.dart:120`；
  `String`：`mall_order.dart:79`、`wallet_api.dart:106`；`int` 分：`wallet_models.dart:21`、`membership_models.dart:40`）
  → 服务端**单方**收敛会打破其中一侧；ES `amount` mapping 为 `double`（`app/internal/transaction/repo_es.go:48`）
  → 改单位**必然 reindex**；`wys_used_car_order.amount` 与 `wys_membership_order.amount_fen` 被
  `migrations/20260924110000_align_column_drift.up.sql:103,66` 放宽为可空，而 Go 侧是非指针类型 → **NULL 与 0 不可区分**。
- 解除条件：`harden-h6b-transaction-money-unit` 完成 `transactions` 五步迁移（加列→回填→双写→切读→删列，
  见 audit §6.2）；**回填前置硬门槛**是 `SELECT count(*) FROM transactions WHERE amount <> round(amount, 2)` 为 0 行
  （audit §5.1：**worker 沙箱未跑**，2026-11-06 由 **coordinator** 只读补跑 = 0 行，仅覆盖本机 `my_go_study`；
  **h6b 需自己重跑并留证据**，不得引用他人绿灯）。其余链路（used_car / auto_finance / mall 改价入口）
  各自单独立 Slice。

## 29. `provider=supabase` 路径金额口径不迁移（双 provider 不一致）

- 文件:行：`app/internal/transaction/repo_supabase.go:137`（Create `payload["amount"] = input.Amount`）、
  `:174`（Update）、`:128`/`:159`（`WithUserToken`）
- status：`Degraded`
- note：这两处把 float 金额直写**云端表**，本仓 `migrations/` 管不到它（`parity-p0-1` 定的 schema 单一真相只覆盖本地 PG）。
  h6 审计给出的结论是**「该路径本轮不做迁移」作为一种合法结论被显式写出**，而不是留空：
  云端加列需 Supabase 侧 migration（另一套发布流程与权限），且代码必须做到「本地路径生效、Supabase 路径行为不变」。
  → 由此产生**已知不对称**：同一用户在 `provider=local` 下金额可精确到分，在 `provider=supabase` 下仍可存 >2dp 脏值。
- 影响面：本仓默认 provider 现为 local，但 `docs/local-auth-postgres.md` 支持回切 supabase；
  一旦回切，`amount_fen` 在云端不存在会**直接写失败**（不是降级，是 500）。
  仍严格 `WithUserToken` 走 RLS，未用 service_role 绕过（`.harness/rules/auth-token-hygiene.md`）。
- 解除条件：Supabase 云端表补 `amount_fen` 列 + RLS/PostgREST schema 缓存同步（`NOTIFY pgrst`）；
  或代码显式禁止「本地已迁移 + provider=supabase」这一组合并给出可读错误。

## 30. mall 后台改价 `normalizeMoney` 静默四舍五入

- 文件:行：`app/internal/mall/service.go:106`（调用）→ `:816-824`（实现）→ `:825-827`（`formatMoney = r.FloatString(2)`）
- status：`Degraded`
- note：这是 h6 审计穷举跨界换算点后找到的**唯一一处「吃掉精度且无任何信号」**的资金写入点：
  用 `big.Rat` 精确解析运营提交的 price 字符串，再 `FloatString(2)` **四舍五入**落库，
  既不报错也不告警（对比：`ParseYuanToFen` 同场景是直接 `return err`）。
  因此 `12.345` 会被静默改成 `12.35`，而订单侧走的是拒绝式 → 两侧行为不一致。
  该函数**只影响 SKU 目录价**（`wys_mall_sku.price`），不影响已生成订单（下单聚合走 `big.Rat` 全程精确，
  `mall/service.go:303,342,346,347`，落库才 `FloatString(2)`）。
- 影响面：运营侧改价可能与实际展示价差 0.5 分级别；不造成用户端多扣款（订单金额由快照价算出）。
  属「无信号的静默变更」，故登记 `Degraded` 而非忽略。
- 解除条件：改为**拒绝式**（>2dp 返回错误）或显式记录一次 `Warn` + 审计日志；
  h6 Brief「不做」明令禁止「顺手加个校验」→ **单独立 Slice**。

## 31. 两种库形态的 `amount_fen` 形状不同（迁移 vs AutoMigrate）

- 文件:行：`migrations/20261106170000_transaction_amount_fen.up.sql`（`ADD COLUMN … bigint` → `ALTER COLUMN … SET NOT NULL` →
  `ADD CONSTRAINT transactions_amount_fen_nonneg CHECK (amount_fen >= 0)`）
  vs `model/entity/transaction.go:25`（`AmountFen *int64` + `gorm:"column:amount_fen"`，GORM 只能建成**可空、无 CHECK**）
  入口分叉在 `app/internal/schemaboot/boot.go:40`（`auto_migrate=true` → AutoMigrate）与 `:52,90`（false → 只校验迁移版本）。
- status：`Degraded`
- note：h6b 只把约束落在 `migrations/`（parity-p0-1 的单一真相），AutoMigrate 走的是 entity 形状，因此**同一段代码在两类库上
  拿到的列定义不同**：迁移库 = `bigint NOT NULL` + `CHECK (amount_fen >= 0)`；AutoMigrate 库 = `bigint` 可空且无任何 CHECK
  （实测 `\d transactions`：`amount_fen|bigint|YES`、约束列表为空、`schema_migrations` 关系不存在）。
  这不会让读路径静默读成 0 元：`AmountFen` 是 `*int64`，`applyAmountFromFen`（`repo_postgres.go:30-31`）在 `nil` 时**直接返回**，
  保留旧列 `amount` 的值（`money_fen_integration_test.go` 的 `TestMoneyUnitAutoMigrateShapeFallsBack` 断言读到 `88.50` 而非 `0`）。
  真正的差额是**写入侧约束**：AutoMigrate 库上没有 NOT NULL、也没有非负 CHECK，脏值/负数只有 Go 侧的拒绝式换算拦着。
- 影响面：`database.auto_migrate=true` 仅出现在 lab/dev 类环境（`config.NoDDLAppEnv` 对 `lan|prod` 强制 false），
  正式环境不会落到无约束形态；但本地/试验库上「SQL 直写负数」不会被拒。
- 解除条件：把 `amount_fen` 的 NOT NULL/CHECK 视为**迁移专属契约**并在启动日志上显式声明当前形态（例如 AutoMigrate 路径
  打一条 `Warn(status=degraded)`），或为 AutoMigrate 补一条 post-migrate 的 DDL 后置钩子。两者都超出 h6b 白名单 → 单独立 Slice。

## 32. 亚分金额（>2dp）被拒后返回 502 而不是 400

- 文件:行：`app/internal/transaction/repo_postgres.go:17`（`ErrAmountPrecision`）→ `:21`/`:112`/`:157`（换算失败上抛）
  → `app/internal/transaction/handler.go:176`（Create：`response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())`）
  → `:202`（Update：`response.BackendError(c, http.StatusBadGateway, err.Error())`）
- status：`Degraded`
- note：h6b 把「>2 位小数」从**静默舍入**改成**拒绝式**，语义正确；但 handler 对仓储返回的错误**一律映射成 5xx**
  （它原先要表达的是「后端 Supabase/PG 不可用」，没有按错误种类分流）。实测（`APP_ENV=dev` + `:8081` + `p06b_boot`）：
  `POST /api/v1/transactions` `amount=1.005` → `http=502`、body `{"error":"金额最多两位小数: 1.005","code":50000}`。
  客户端输入错误被报成服务端故障：Flutter 侧若按 5xx 做重试/降级提示，会对一个永远不可能成功的请求重试。
  `app/internal/transaction/handler.go` 在 h6b **白名单外**（Brief 只允许 migrations / entity / repo_postgres / service / 测试 / schemaboot / 文档），
  且 `transaction_request.go` 在**黑名单**内 → 本轮刻意不顺手改。
- 影响面：不影响资金正确性（写入被拒、库内 0 脏值，漂移核对 0 行），只影响错误语义与客户端体验。
- 解除条件：`errors.Is(err, transaction.ErrAmountPrecision)` → `400` + 参数错误码（一处 handler 错误映射改动），
  或在 DTO 层用 `entity.ParseYuanToFen` 校验字符串后再进仓储。单独立 Slice。

---

## 本轮语义变更摘要（P0-5）

| 位置 | 改前 | 改后 |
|------|------|------|
| `membership/handler.go` 华为回调 | 200 `{"accepted":true}` | 501 + 业务码 `10005`，文案说明不受理 |
| `delivery/http/dto/response/common.go` | 无「未实现」码 | 新增 `CodeNotImplemented = 10005` |
| `delivery/worker/handler.go` SMS | `Info("sms task stub…")` | `Warn("sms task NOT IMPLEMENTED … acked and dropped")` |
| `short_video/service.go` 媒体 | 无条件填充样例 URL | `demoMedia` 守卫；正式环境缺 URL → 400 |
| `community/service.go` 媒体 | 无条件填充 picsum/样例 | `requiresRealMedia` + `demoMedia` 守卫；正式环境 → 400 |
| `cmd/api/main.go` 种子 | 10 处无条件自动执行 | `seedIfLab(labSeeds,…)`，正式环境跳过并记日志 |
| `cmd/api/main.go` SSE 回退 | 仅「无 key」一条 Warn；默认分支静默 mock | 两条路径都 `Warn(status=degraded)` |
| `repository/llm/mock_stream_provider.go` | 无任何可观测 | `degraded_path_total{name="llm_mock"}` |
| `usecase/cs_agent_usecase.go` | 无任何可观测 | `degraded_path_total{name="cs_agent_template"}` |
| `common/metrics/metrics.go` | 无降级路径指标 | 新增 `DegradedPathsTotal` CounterVec |

## 后续轮次语义变更摘要（P0-2 支付流水与回调）

| 位置 | 改前 | 改后 |
|------|------|------|
| `payment/service.go` `Prepay` | `out_trade_no` 生成即丢，无仓储 | 先落 `wys_payment_records(pending)` 再问渠道；未到达渠道才置 failed |
| `payment/router.go` | 仅 `POST /payments/prepay` | 新增公开回调 `POST /payments/notify/{wechat,alipay}`（不进 SessionAuth，靠验签） |
| `payment/verify.go` | 无回调验签 | 微信 v2 MD5 + 支付宝 RSA2 全报文校验；商户号/app_id/金额/渠道四项对齐流水；缺密钥一律拒 |
| `payment/settle.go` | 不存在 | 验签通过后**流水 + 业务发货同事务**；重复投递只核销 1 次 |
| `mall/service.go` `PayOrder`（微信/支付宝） | 本地直接标记已付、无流水 | 返回 `{status:"pending", out_trade_no, prepay_channel, prepay_params}`，推进只由回调/查单驱动 |
| `membership/service.go` `ConfirmBuyout` | 回跳即发权益 | 需已核销流水（`ConfirmRefPaid`）；payment 未装配直接拒绝 |
| `delivery/worker/handler.go` | 无支付任务 | 新增 `payment.query_pending` / `payment.reconcile`；未装配 payment 时**返回 error 走重投→DLQ** |
| `core/queue/payment_ticker.go` | 不存在 | 定时只入队（`core` 不依赖 `app`），对账同日不重复入队 |
| `deploy/observability/alerts.yml` | 无支付告警 | 新增 `payment` 组 4 条（验签失败、pending 卡住、任务未跑、对账差异） |

## 后续轮次语义变更摘要（membership 资金事务化 follow-up）

登记于 P0-3 验收记录「已知残留」，本轮（2026-11-04）执行；改法与 P0-3 的 `PayOrder` 完全同形。

| 位置 | 改前 | 改后 |
|------|------|------|
| `membership/repository.go` `Repository` | 没有事务入口，跨表写只能各提交各的 | 新增 `Transaction(ctx, fn)`；`repo_postgres.go` 落到 `txdb.Begin` |
| `membership/service.go` `completeBalanceBuyout` | 扣款先独立提交，失败后 `_, _ = u.walletSvc.Credit(..., ":rollback")` 补偿 —— 补偿本身失败无人知晓，等于静默吞钱 | 扣款 + 改单 + 发权益同事务，任一步失败整体回滚；两处补偿删除 |
| `membership/service.go` `ConfirmBuyout`（渠道分支） | `MarkOrderPaid` 与 `ExtendEntitlement` 两次独立写，可留「单已付、权益没发」 | 两次写并入同一事务；`ErrOrderPaid` 改到**事务外**再递归确认（事务里递归会白滚一遍） |
| `membership/repo_postgres.go` | 写路径 `r.db.WithContext(ctx)` | `txdb.DB(ctx, r.db).WithContext(ctx)`；`ExtendEntitlement` 自带的内层事务嵌在外层时落 GORM SAVEPOINT |
| `membership/service_test.go` 夹具 | 无 `Transaction` | `memMembershipRepo.Transaction` 用快照/还原**假演**原子性（与 mall 夹具同构），真实语义由 `-tags=integration` 用例承担 |

## 复核方法

```bash
# 指标可见性（local/dev 启动后）
curl -s http://127.0.0.1:8080/metrics | grep degraded_path_total

# 华为回调不再返回成功语义
curl -s -X POST http://127.0.0.1:8080/api/v1/membership/huawei/notifications \
  -H "authorization: <session token>" | jq .          # 期望 HTTP 501 + code 10005

# 正式环境固定码不可登录（单元层）
go -C app test ./internal/usecase/... -run 'OTP'
# 正式环境固定码不可登录（实机）
APP_ENV=prod make run   # 再调 POST /api/v1/user/phone/otp/verify {13400000000,123456} → 400 短信登录暂未开放
# h7 更正（2026-11-07）：`APP_ENV=production` 现由 config.Load 归一化到 `prod`，
# 自动加载 configs/config.prod.yaml（release / sslmode=require / auto_migrate=false），可以启动。
# 未知环境名或缺失覆盖文件一律拒绝启动并点名文件，不再静默回退 config.yaml（debug）。

# 清单抽查：本文件任取 3 条，按「文件:行」打开对照行为
```

## h7 追加（2026-11-07）：环境名与 /metrics 收口

| 位置 | 之前 | 现在 |
|------|------|------|
| `common/config/config.go` `Load` | `APP_ENV=production` 找不到 `config.production.yaml` 时 Warn 后按 `config.yaml`（debug / sslmode=disable）继续跑 | 归一化 `production`→`prod` 加载 `config.prod.yaml`；正式/未知/拼错环境缺覆盖文件即拒绝启动并点名文件 |
| `app/internal/delivery/http/router/router.go` `/metrics` | 随 `8080:8080` 暴露到局域网 | 只放行 loopback + `METRICS_ALLOW_CIDRS`（Docker 内网抓取），局域网 IP 403 |

证据：`docs/acceptance-records/2026-11-05-harden-h7-prod-bootability.md`。

## 与 Program checklist 的交叉核对

- [x] 本清单条目 1–17 覆盖 `plans/epics/xhs-parity-p0-correctness.md` 假实现行点名的全部路径
      （华为回调 / SMS / OTP / mock 媒体 / LLM mock / 客服 Agent / 演示种子 / 占位邮箱 / JPush），
      并补入 `ConfirmBuyout`（epic 指向 P0-2）。
- [x] P0-4 结果已登记（条目 14 `Removed` + 条目 15 `Deferred`），未声称机审完成。
- [x] 未登记即宣称完成的路径：本次核对未发现新增项（Program 里 P0-1/P0-2/P0-3/P0-6/P0-7 均已在各自
      Epic/Slice 下跟踪，本文件只做指向）。

---

## 本轮新增条目（harden-h5 静默吞错留痕，2026-11-05）

> 来源：Slice `harden-h5-silent-error-sweep`。这 5 条不是「假实现」，而是「静默吞掉的失败」：
> 原代码用 `_, _ =` / `_ = err` 丢弃返回值，失败无痕迹。按 R4 口径 ——
> **默认保持降级 + 打 `degraded_path_total{name=...}` 指标**，只有会丢数的那条升级为硬失败。
> 下表 `name` 即代码里 `metrics.DegradedPathsTotal.WithLabelValues("<name>")` 的字面量，一一对应。

| # | 文件:行（改前） | name | 处置 |
|---|----------------|------|------|
| 23 | `delivery/worker/handler.go:216` | `push_dedup_del_failed` | 删键失败**参与返回 error**（两个 `%w`，`errors.Is` 可命中）+ `h.log.Error` + 指标 |
| 24 | `usecase/realtime_push_usecase.go:168` | `offline_push_failed` | 保持 WS 投递成功，指标留痕 |
| 25 | `access/service.go:102` | `access_store_group_sync_failed` | 成员入库不回滚，指标留痕 |
| 26 | `usecase/profile_usecase.go:82` | `profile_store_group_sync_failed` | 切店成功不回滚，指标留痕 |
| 27 | `usecase/im_group_usecase.go:163` | `im_group_create_failed` | 继续 join + upsert，指标留痕 |

### 23. 定时广播删去重键失败

- 文件:行：`app/internal/delivery/worker/handler.go:216`（改后为 `cleanupDedupAfterEnqueueError`）
- status：`Degraded`（升级为硬失败）· note：`SetNX` 抢占去重键后入队失败，原 `_, _ = Del` 试图回滚；
  删键也失败则键残留 → 该用户该时段**永久跳过推送**（丢数）且无人知。现 Del 失败 `h.log.Error` +
  `push_dedup_del_failed` +1，并把 enqueue/del 两个 error 一起返回，交既有重试/死信，不静默留坏键。
- 影响面：改动前「入队失败 + 删键失败」静默丢一条定时推送；改后为可见失败与告警。
- 解除条件：不需要（本条已收口为硬失败）；去重键自愈属 P0-6 重试/死信范畴。

### 24. 离线推送失败静默

- 文件:行：`app/internal/usecase/realtime_push_usecase.go:168` · status：`Degraded`
- note：`DeliverPush` 写 Redis 事件并广播 WS 后，离线通道失败原本 `_, _ =` 丢弃。现 `if err != nil` 时
  `offline_push_failed` +1，返回仍成功（WS 不受影响）。usecase 层无 zap logger、本轮不新增构造参数
  （避免与 h8 改 `main.go` 冲突），留痕只走指标，日志留 Deferred。
- 影响面：离线用户仍可能收不到，但 `/metrics` 可计数，不再是「无人知」。
- 解除条件：接真实离线通道 + 重试（P2）；或注入 logger 补日志。

### 25. 加成员后拉店群失败静默

- 文件:行：`app/internal/access/service.go:102` · status：`Degraded`
- note：`UpsertMember` 成员已入库后调 `EnsureStoreMembership` 拉群，失败原本 `_, _ =` 丢弃 → 数据不一致。
  现 `if err != nil` 时 `access_store_group_sync_failed` +1；成员入库不回滚（保持降级）。
- 影响面：成员在店内但可能不在店群，指标可见。· 解除条件：拉群改可靠异步补拉（P2-8 outbox，本轮不做）。

### 26. 切店后拉店群失败静默

- 文件:行：`app/internal/usecase/profile_usecase.go:82` · status：`Degraded`
- note：与条目 25 同形状（切店成功、拉群失败），入口不同故 name 分开便于定位。现 `if err != nil` 时
  `profile_store_group_sync_failed` +1，切店结果不回滚。· 影响面/解除条件：同条目 25。

### 27. 门店群 `CreateGroup` 失败被 `_ = err` 吞掉

- 文件:行：`app/internal/usecase/im_group_usecase.go:163` · status：`Degraded`
- note：`EnsureStoreMembership` 建群时 `CreateGroup` 失败被 `_ = err` 吞掉，注释称「群可能已存在」，
  但网络/凭据错误也走同一分支。现该分支 `im_group_create_failed` +1，仍继续 join + upsert（保持降级）。
  **指标无法区分「已存在」与「真失败」**，这是刻意保留的降级口径。
- 影响面：真失败时群未建成但本地已 upsert，靠 join 补救；指标可计数但不分类。
- 解除条件：融云客户端按错误码区分「已存在」与其它错误再分别处置（P2）。

## 本轮语义变更摘要（harden-h5 静默吞错）

| 位置 | 改前 | 改后 |
|------|------|------|
| `delivery/worker/handler.go:216` | `_, _ = Del` 吞掉删键失败 | 抽 `cleanupDedupAfterEnqueueError`：Del 失败 → `h.log.Error` + `push_dedup_del_failed` +1 + 参与返回 error |
| `usecase/realtime_push_usecase.go:168` | `_, _ = NotifyUser` | 降级 + `offline_push_failed` +1 |
| `access/service.go:102` | `_, _ = EnsureStoreMembership` | 降级 + `access_store_group_sync_failed` +1 |
| `usecase/profile_usecase.go:82` | `_, _ = EnsureStoreMembership` | 降级 + `profile_store_group_sync_failed` +1 |
| `usecase/im_group_usecase.go:163` | `_ = err` | 降级 + `im_group_create_failed` +1 |
