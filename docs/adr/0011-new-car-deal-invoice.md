# 新车成交发票：独立表 + 本人当前店范围；图只本地预览

新车成交是销售在当前店为自己提交的成交发票业务，与店务审核单、商城订单分离。复用并扩展 `wys_store_customer`（加必填 `phone`）作购车客户；新建 `wys_deal_invoice` 存发票单。本切片提供列表/摘要/详情/提交/同单重提；**不做**审核通过/驳回 API、真实图片上传、评分写接口。客户端选相册/拍摄仅本地展示；`image_url` 可空。列表与统计仅 `uploader_user_id = session` 且 `store_id = current_store`。Flutter 包暂留 `module_settings/deal_invoice`，首页入口改「新车成交」并跳现有路由。
