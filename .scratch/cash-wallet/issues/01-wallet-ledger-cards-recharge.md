# 01 — 钱包账本 + 绑卡 + 充值 + Flutter 钱包页

**What to build:** 已登录用户从「我的」快捷服务进入「我的钱包」，能看到人民币余额与流水、绑定/列表/默认/删除本地模拟银行卡，并能用自由金额（0.01–50000）经模拟渠道（绑卡/支付宝/微信）充值入账。本票不做商城余额支付与退款入账。

**Blocked by:** None — can start immediately

**Status:** done

- [x] 现金钱包与流水平久化（`wys_` 前缀）；账号全局余额；SessionAuth API 可查余额/流水、绑卡 CRUD、POST 充值
- [x] 充值金额边界校验；成功后余额与流水一致；不存完整卡号；默认卡至多一张
- [x] Wallet usecase 缝有自动化测试锁住充值、边界、默认卡规则
- [x] Flutter：`features/wallet`；我的「我的钱包」进入真页面（不再 toast）；未登录引导登录；页内可看余额/流水、管卡、充值
- [x] 与积分 UI/API 分离，不复用积分表
