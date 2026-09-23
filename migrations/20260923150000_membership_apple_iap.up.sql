-- +goose Up
ALTER TABLE wys_membership_entitlement
  ADD COLUMN IF NOT EXISTS apple_original_transaction_id text NOT NULL DEFAULT '';

ALTER TABLE wys_membership_order DROP CONSTRAINT IF EXISTS chk_wys_membership_order_channel;
ALTER TABLE wys_membership_order ADD CONSTRAINT chk_wys_membership_order_channel
  CHECK (channel IN ('wechat', 'alipay', 'balance', 'huawei', 'apple'));
