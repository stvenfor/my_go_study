-- 用户收货地址簿。无物理外键。

CREATE TABLE IF NOT EXISTS wys_user_address (
  address_id      bigserial PRIMARY KEY,
  user_id         varchar(64) NOT NULL,
  receiver_name   text NOT NULL,
  receiver_phone  varchar(20) NOT NULL,
  province        text NOT NULL DEFAULT '',
  city            text NOT NULL DEFAULT '',
  district        text NOT NULL DEFAULT '',
  detail_address  text NOT NULL,
  postal_code     varchar(16) NOT NULL DEFAULT '',
  is_default      boolean NOT NULL DEFAULT false,
  label           text NOT NULL DEFAULT '',
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  deleted_at      timestamptz,
  CONSTRAINT chk_wys_user_address_name CHECK (char_length(btrim(receiver_name)) > 0),
  CONSTRAINT chk_wys_user_address_phone CHECK (char_length(btrim(receiver_phone)) >= 6),
  CONSTRAINT chk_wys_user_address_detail CHECK (char_length(btrim(detail_address)) > 0)
);

CREATE INDEX IF NOT EXISTS ix_wys_user_address_user
  ON wys_user_address (user_id, address_id DESC)
  WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_user_address_default
  ON wys_user_address (user_id)
  WHERE is_default = true AND deleted_at IS NULL;

COMMENT ON TABLE wys_user_address IS '用户收货地址簿。订单只快照字段，不引用本表 PK';
