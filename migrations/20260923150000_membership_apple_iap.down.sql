-- +goose Down
ALTER TABLE wys_membership_order DROP CONSTRAINT IF EXISTS chk_wys_membership_order_channel;
ALTER TABLE wys_membership_order ADD CONSTRAINT chk_wys_membership_order_channel
  CHECK (channel IN ('wechat', 'alipay', 'balance', 'huawei'));
ALTER TABLE wys_membership_entitlement DROP COLUMN IF EXISTS apple_original_transaction_id;
