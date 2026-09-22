-- 购车客户手机号 + 新车成交发票
ALTER TABLE wys_store_customer
  ADD COLUMN IF NOT EXISTS phone varchar(32) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS wys_deal_invoice (
  invoice_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  uploader_user_id varchar(64) NOT NULL,
  customer_id bigint NOT NULL,
  customer_phone varchar(32) NOT NULL,
  customer_name varchar(128) NOT NULL DEFAULT '',
  status smallint NOT NULL DEFAULT 0,
  image_url text,
  reject_reason text,
  rating_stars smallint,
  submitted_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_deal_invoice_status CHECK (status IN (0, 1, 2, 3)),
  CONSTRAINT chk_wys_deal_invoice_rating CHECK (
    rating_stars IS NULL OR (rating_stars BETWEEN 1 AND 5)
  )
);

CREATE INDEX IF NOT EXISTS idx_wys_deal_invoice_uploader_submitted
  ON wys_deal_invoice (store_id, uploader_user_id, submitted_at DESC);

CREATE INDEX IF NOT EXISTS idx_wys_deal_invoice_uploader_status
  ON wys_deal_invoice (store_id, uploader_user_id, status, submitted_at DESC);

COMMENT ON TABLE wys_deal_invoice IS '新车成交发票。0=待审 1=已通过待评 2=已评 3=未通过';
