-- +goose Down
ALTER TABLE wys_mall_order DROP CONSTRAINT IF EXISTS chk_wys_mall_order_channel;
ALTER TABLE wys_mall_order ADD CONSTRAINT chk_wys_mall_order_channel
  CHECK (payment_channel IS NULL OR payment_channel IN (1, 2, 3, 4, 5));
ALTER TABLE wys_mall_payment DROP CONSTRAINT IF EXISTS chk_wys_mall_payment_channel;
ALTER TABLE wys_mall_payment ADD CONSTRAINT chk_wys_mall_payment_channel
  CHECK (payment_channel IN (1, 2, 3, 4, 5));
ALTER TABLE wys_mall_refund DROP CONSTRAINT IF EXISTS chk_wys_mall_refund_channel;
ALTER TABLE wys_mall_refund ADD CONSTRAINT chk_wys_mall_refund_channel
  CHECK (payment_channel IN (1, 2, 3, 4, 5));
DROP TABLE IF EXISTS wys_bank_card;
DROP TABLE IF EXISTS wys_cash_ledger;
DROP TABLE IF EXISTS wys_cash_wallet;
