# 二手车：独立业务单表 + 列表/详情/新建；与收支账本拆开

「二手车」入口不再挂 `transactions` 个人收支。新建 `wys_used_car_order`（前缀 `/api/v1/used-car-orders`）承载店务单据：类型置换/专卖/收车；审核态对齐成交发票四态；客户复用 `wys_store_customer`；车况字段（车型名、车牌、VIN、里程、年款）与业务金额必填；可选 `image_url` 占位、只读 `rating_stars` / `reject_reason`。第一期提供摘要/列表（可筛 kind+status）/详情/新建（默认待审核）及客户选择器；不做审核写、重提、评价写、真实上传。个人收支改由「我的」入口「收支」承接原 transactions UI。首页待办新增小卡类型 `used_car_pending_review`（`store_admin`、全店待审计数），与 `order_pending_review`（店务审核单）分离。

Rejected: 继续把二手车当成收支账本美化；把业务单塞进 `wys_store_review_order` 或 `wys_mall_order`；硬改 `transactions` 加状态/车况。
