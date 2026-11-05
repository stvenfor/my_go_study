-- 渠道支付流水 + 对账差异（P0-2）
-- out_trade_no 唯一是回调幂等的锚点；金额整数分，禁止 float。
CREATE TABLE IF NOT EXISTS wys_payment_records (
  id bigserial PRIMARY KEY,
  out_trade_no varchar(64) NOT NULL,
  user_id varchar(64) NOT NULL,
  channel varchar(16) NOT NULL,
  amount_fen bigint NOT NULL,
  status smallint NOT NULL DEFAULT 0,
  ref_type varchar(32) NOT NULL,
  ref_id varchar(64) NOT NULL,
  channel_trade_no varchar(64),
  -- text 而非 jsonb：微信回调是 XML、支付宝是 form 表单，jsonb 会直接拒收（SQLSTATE 22P02）让核销永远失败。
  notify_raw text,
  fail_reason varchar(128) NOT NULL DEFAULT '',
  paid_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_payment_records_status CHECK (status IN (0, 1, 2)),
  CONSTRAINT chk_wys_payment_records_channel CHECK (channel IN ('wechat', 'alipay')),
  CONSTRAINT chk_wys_payment_records_amount CHECK (amount_fen > 0)
);

-- 商户单号唯一 = 回调幂等的锚点。建成**索引**而不是表级约束：
-- 名字要与 entity 的 uniqueIndex 一致，lab 环境 AutoMigrate 才不会去找 uni_* 而报错。
CREATE UNIQUE INDEX IF NOT EXISTS uk_wys_payment_records_out_trade_no
  ON wys_payment_records (out_trade_no);

-- 查单兜底按 (status, created_at) 扫超时 pending；对账按时间窗扫。
CREATE INDEX IF NOT EXISTS idx_wys_payment_records_status_created
  ON wys_payment_records (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wys_payment_records_ref
  ON wys_payment_records (ref_type, ref_id);
CREATE INDEX IF NOT EXISTS idx_wys_payment_records_user_created
  ON wys_payment_records (user_id, created_at DESC);

COMMENT ON TABLE wys_payment_records IS '渠道支付流水：prepay 落库 → notify/查单核销；金额整数分';

-- T+1 对账差异：只增不改（resolved 由人工/后续工单置位）。
CREATE TABLE IF NOT EXISTS wys_payment_reconcile_diff (
  id bigserial PRIMARY KEY,
  recon_date varchar(10) NOT NULL,
  channel varchar(16) NOT NULL,
  out_trade_no varchar(64) NOT NULL DEFAULT '',
  kind varchar(32) NOT NULL,
  local_status smallint NOT NULL DEFAULT 0,
  local_amount_fen bigint NOT NULL DEFAULT 0,
  channel_status varchar(32) NOT NULL DEFAULT '',
  channel_amount_fen bigint NOT NULL DEFAULT 0,
  channel_trade_no varchar(64) NOT NULL DEFAULT '',
  resolved boolean NOT NULL DEFAULT false,
  resolve_note varchar(256) NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_payment_reconcile_diff_kind
    CHECK (kind IN ('local_missing', 'channel_missing', 'amount_mismatch', 'status_mismatch'))
);

CREATE INDEX IF NOT EXISTS idx_wys_payment_reconcile_diff_date
  ON wys_payment_reconcile_diff (recon_date, resolved);

-- 对账任务至少投一次（worker 重启 / 人工补跑 / 死信重放都可能重复），
-- 靠唯一键 + ON CONFLICT DO NOTHING 保证同一笔同一类差异只有一条。
CREATE UNIQUE INDEX IF NOT EXISTS uk_wys_payment_reconcile_diff
  ON wys_payment_reconcile_diff (recon_date, out_trade_no, kind);

COMMENT ON TABLE wys_payment_reconcile_diff IS 'T+1 对账差异；alerts.yml 按未解决差异数告警';
