-- 故意不回滚：wys_payment_records / wys_payment_reconcile_diff 是资金事实与差异留痕，
-- down 迁移若 DROP 会销毁对账依据。回滚只回滚代码，表保留（与 20260924110000 同样处理）。
SELECT 1;
