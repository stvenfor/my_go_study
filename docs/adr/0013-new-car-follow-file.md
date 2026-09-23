# 新车跟进档案：独立表 + ABEH 级别；本人当前店

新车跟进是销售在当前店维护的意向客户档案，与成交发票、店务审核单分离。复用 `wys_store_customer`；新建 `wys_new_car_follow_file`。跟进级别仅 `A/B/E/H`，读模型派生购车意向档高/中/低（H/A→高，B→中，E→低）。写下次跟进时间时回写客户 `next_follow_up_at`。C1 提供 summary/列表/建档/详情/PATCH；不做流水、店管全店、审核。
