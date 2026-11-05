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
| 18 | `membership_order` 无业务核销器（权益需回跳/查单触发；两次写已同事务） | Degraded | 待排（`RegisterSettler`） |
| 19 | 对账只做「渠道查单」方向，`local_missing` 未实现 | Deferred | 待排（渠道账单文件下载） |
| 20 | 商城 Apple/华为 IAP 渠道仍本地标记已付、无渠道流水 | Placeholder | 待排（IAP 专项） |
| 21 | `unifiedorder` 响应未验签（只信回调报文） | Degraded | 待排（P2 支付加固） |

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

## 18. `membership_order` 没有业务核销器

- 文件:行：`app/internal/payment/settle.go:103-116`（`settled_no_settler` 分支）
- status：`Degraded`
- note：渠道回调只把流水置 paid 并打 Warn（`ref_type` 未登记核销器），权益由会员侧 `ConfirmRefPaid`
  （回跳同步查单）或下一次查单任务补齐。指标上可观测，不是静默消失。
  会员侧内部两次写（`MarkOrderPaid` → `ExtendEntitlement`）**已经**在同一 `txdb` 事务里
  （`membership/repository.go:Transaction`，P0-3 同形态），所以「半状态」这条已不成立；
  剩下的只是没注册 Settler —— 即**权益仍要客户端/查单来触发**，回调不会主动发。
- 影响面：只走「客户端回跳 + 查单」这条链发权益；若用户回跳后再也不请求，权益延迟到下一次 pending 查单
  （`scheduler.payment.query_interval_minutes`）。
- 解除条件：`pay.RegisterSettler(entity.PaymentRefMembershipOrder, memSvc)`，让回调直接发权益
  （仓储的 `Transaction` 已就绪，改法与 mall 的 `SettleChannelPaid` 同形）。

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
# 注意：`APP_ENV=production` 当前**无法启动** —— config.Load 要求 configs/config.production.yaml，
# 仓库只有 config.{,dev,lan,prod}.yaml。守卫语义（IsLabAppEnv）对 prod/production 一致。

# 清单抽查：本文件任取 3 条，按「文件:行」打开对照行为
```

## 与 Program checklist 的交叉核对

- [x] 本清单条目 1–17 覆盖 `plans/epics/xhs-parity-p0-correctness.md` 假实现行点名的全部路径
      （华为回调 / SMS / OTP / mock 媒体 / LLM mock / 客服 Agent / 演示种子 / 占位邮箱 / JPush），
      并补入 `ConfirmBuyout`（epic 指向 P0-2）。
- [x] P0-4 结果已登记（条目 14 `Removed` + 条目 15 `Deferred`），未声称机审完成。
- [x] 未登记即宣称完成的路径：本次核对未发现新增项（Program 里 P0-1/P0-2/P0-3/P0-6/P0-7 均已在各自
      Epic/Slice 下跟踪，本文件只做指向）。
