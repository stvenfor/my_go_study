CREATE TABLE IF NOT EXISTS wys_cash_wallet (
  user_id     text PRIMARY KEY,
  balance_fen bigint NOT NULL DEFAULT 0 CHECK (balance_fen >= 0),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wys_cash_ledger (
  ledger_id   bigserial PRIMARY KEY,
  user_id     text NOT NULL,
  delta_fen   bigint NOT NULL,
  balance_fen bigint NOT NULL,
  reason      text NOT NULL,
  ref_id      text NOT NULL DEFAULT '',
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_wys_cash_ledger_user
  ON wys_cash_ledger (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS wys_bank_card (
  card_id     bigserial PRIMARY KEY,
  user_id     text NOT NULL,
  bank_name   text NOT NULL,
  card_last4  text NOT NULL,
  holder_name text NOT NULL DEFAULT '',
  is_default  boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_wys_bank_card_user ON wys_bank_card (user_id);

-- 商城支付渠道扩展余额(6)
ALTER TABLE wys_mall_order DROP CONSTRAINT IF EXISTS chk_wys_mall_order_channel;
ALTER TABLE wys_mall_order ADD CONSTRAINT chk_wys_mall_order_channel
  CHECK (payment_channel IS NULL OR payment_channel IN (1, 2, 3, 4, 5, 6));
ALTER TABLE wys_mall_payment DROP CONSTRAINT IF EXISTS chk_wys_mall_payment_channel;
ALTER TABLE wys_mall_payment ADD CONSTRAINT chk_wys_mall_payment_channel
  CHECK (payment_channel IN (1, 2, 3, 4, 5, 6));
ALTER TABLE wys_mall_refund DROP CONSTRAINT IF EXISTS chk_wys_mall_refund_channel;
ALTER TABLE wys_mall_refund ADD CONSTRAINT chk_wys_mall_refund_channel
  CHECK (payment_channel IN (1, 2, 3, 4, 5, 6));
