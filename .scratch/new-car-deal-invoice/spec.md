# New Car Deal Invoice

**Status:** done

用语：`CONTEXT.md` §新车成交  
方案：[docs/deal-invoice-api.md](../../docs/deal-invoice-api.md)  
ADR：[docs/adr/0011-new-car-deal-invoice.md](../../docs/adr/0011-new-car-deal-invoice.md)

实现按 `issues/` 阻塞边，从 frontier 抓票；每票独立会话 `/implement`。跨仓：Go BFF 本仓库；Flutter 在 `my_ai_project`（`module_settings/deal_invoice`，本 feature 不迁包）。

| # | Ticket | Blocked by |
|---|--------|------------|
| 01 | [首页「新车成交」：真摘要 + 真列表](./issues/01-summary-and-list.md) | — |
| 02 | [上传页选择购车客户](./issues/02-customer-picker.md) | 01 |
| 03 | [新建提交（本地相册/拍摄预览）](./issues/03-create-with-local-image.md) | 02 |
| 04 | [详情 + 驳回同单重提](./issues/04-detail-and-resubmit.md) | 03 |

**Frontier 起点：** 只开 **01**。

```text
01 → 02 → 03 → 04
```
