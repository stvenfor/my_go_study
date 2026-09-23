# 04 — 钱包 API/联调文档 + 验收记录

**What to build:** 补齐钱包与余额支付的可读 API/联调说明，并留下一份短验收记录（本地或 LAN 上：充值 → 余额支付 → 取消退回钱包），方便前后端与 QA 对照。

**Blocked by:** 02 — 商城余额支付 + 人民币退回钱包；03 — Flutter 商城支付选渠道（含余额）

**Status:** done

- [x] `docs/` 下有钱包 API（余额/流水/卡/充值）与商城渠道 6、退回钱包语义说明，并链到 ADR-0015
- [x] 短 acceptance 记录覆盖充值、余额支付、取消入账三条主路径
- [x] Spec 票表状态与文档入口可从 docs 或 `.scratch/cash-wallet/` 找到
