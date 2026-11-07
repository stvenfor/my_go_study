-- 回滚：只撤本迁移加的东西。
-- 旧列 amount 全程未动（双写仍在），所以删 amount_fen 不丢任何数据 —— 这是「旧列不删」这条「不做」换来的回滚能力。
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_amount_fen_nonneg;
ALTER TABLE transactions DROP COLUMN IF EXISTS amount_fen;
