-- +goose Up
CREATE TABLE IF NOT EXISTS wys_membership_entitlement (
  user_id                text NOT NULL,
  tier                   text NOT NULL,
  expires_at             timestamptz NOT NULL,
  source_channel         text NOT NULL DEFAULT '',
  huawei_purchase_token  text NOT NULL DEFAULT '',
  huawei_subscription_id text NOT NULL DEFAULT '',
  updated_at             timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, tier),
  CONSTRAINT chk_wys_membership_tier CHECK (tier IN ('svip', 'ai_svip'))
);

CREATE TABLE IF NOT EXISTS wys_membership_order (
  order_id     bigserial PRIMARY KEY,
  user_id      text NOT NULL,
  tier         text NOT NULL,
  plan_id      text NOT NULL,
  channel      text NOT NULL,
  amount_fen   bigint NOT NULL CHECK (amount_fen > 0),
  status       smallint NOT NULL DEFAULT 0,
  out_trade_no text NOT NULL DEFAULT '',
  created_at   timestamptz NOT NULL DEFAULT now(),
  paid_at      timestamptz,
  CONSTRAINT chk_wys_membership_order_tier CHECK (tier IN ('svip', 'ai_svip')),
  CONSTRAINT chk_wys_membership_order_channel CHECK (channel IN ('wechat', 'alipay', 'balance', 'huawei')),
  CONSTRAINT chk_wys_membership_order_status CHECK (status IN (0, 1))
);
CREATE INDEX IF NOT EXISTS ix_wys_membership_order_user
  ON wys_membership_order (user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS ux_wys_membership_order_out_trade
  ON wys_membership_order (out_trade_no) WHERE out_trade_no <> '';
