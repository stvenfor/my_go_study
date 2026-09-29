-- 商城表。无物理外键。可重复执行。payment_channel: 1 支付宝 2 微信 3 苹果内购 4 华为内购。

CREATE TABLE IF NOT EXISTS wys_mall_category (
  category_id bigserial PRIMARY KEY,
  store_id    integer NOT NULL,
  parent_id   bigint,
  name        text NOT NULL,
  sort        integer NOT NULL DEFAULT 0,
  status      smallint NOT NULL DEFAULT 1,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,
  CONSTRAINT chk_wys_mall_category_status CHECK (status IN (0, 1))
);
CREATE INDEX IF NOT EXISTS ix_wys_mall_category_store
  ON wys_mall_category (store_id) WHERE deleted_at IS NULL;
COMMENT ON TABLE wys_mall_category IS '门店商品类目';

CREATE TABLE IF NOT EXISTS wys_mall_product (
  product_id   bigserial PRIMARY KEY,
  store_id     integer NOT NULL,
  category_id  bigint,
  kind         smallint NOT NULL,
  title        text NOT NULL,
  cover_url    text,
  cover_aspect numeric(4,2) NOT NULL DEFAULT 1.00,
  status       smallint NOT NULL DEFAULT 0,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz,
  CONSTRAINT chk_wys_mall_product_kind CHECK (kind IN (0, 1)),
  CONSTRAINT chk_wys_mall_product_status CHECK (status IN (0, 1, 2)),
  CONSTRAINT chk_wys_mall_product_aspect CHECK (cover_aspect >= 0.50 AND cover_aspect <= 2.00)
);
CREATE INDEX IF NOT EXISTS ix_wys_mall_product_store
  ON wys_mall_product (store_id, status) WHERE deleted_at IS NULL;
COMMENT ON TABLE wys_mall_product IS '商品 SPU。kind 0 实体 1 虚拟';
COMMENT ON COLUMN wys_mall_product.kind IS '0=实体 1=虚拟';
COMMENT ON COLUMN wys_mall_product.status IS '0=草稿 1=在售 2=下架';
COMMENT ON COLUMN wys_mall_product.cover_aspect IS '封面高/宽，瀑布流用';

-- 已有库补列
ALTER TABLE wys_mall_product ADD COLUMN IF NOT EXISTS cover_aspect numeric(4,2) NOT NULL DEFAULT 1.00;

CREATE TABLE IF NOT EXISTS wys_mall_sku (
  sku_id       bigserial PRIMARY KEY,
  product_id   bigint NOT NULL,
  sku_code     varchar(64) NOT NULL,
  title        text NOT NULL DEFAULT '',
  specs        jsonb NOT NULL DEFAULT '{}',
  price        numeric(10,2) NOT NULL,
  stock_qty    integer NOT NULL DEFAULT 0,
  version      integer NOT NULL DEFAULT 0,
  status       smallint NOT NULL DEFAULT 0,
  deliver_type smallint,
  content_url  text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  deleted_at   timestamptz,
  CONSTRAINT chk_wys_mall_sku_price CHECK (price >= 0),
  CONSTRAINT chk_wys_mall_sku_stock CHECK (stock_qty >= 0),
  CONSTRAINT chk_wys_mall_sku_status CHECK (status IN (0, 1)),
  CONSTRAINT chk_wys_mall_sku_deliver CHECK (deliver_type IS NULL OR deliver_type IN (0, 1))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_sku_code
  ON wys_mall_sku (product_id, sku_code) WHERE deleted_at IS NULL;
COMMENT ON TABLE wys_mall_sku IS 'SKU。价格与实体库存在此行';
COMMENT ON COLUMN wys_mall_sku.deliver_type IS '仅虚拟：0=兑换码 1=内容地址';

CREATE TABLE IF NOT EXISTS wys_mall_virtual_code (
  code_id       bigserial PRIMARY KEY,
  sku_id        bigint NOT NULL,
  code          text NOT NULL,
  status        smallint NOT NULL DEFAULT 0,
  order_item_id bigint,
  created_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_mall_virtual_code_status CHECK (status IN (0, 1, 2))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_virtual_code
  ON wys_mall_virtual_code (sku_id, code);
COMMENT ON TABLE wys_mall_virtual_code IS '虚拟商品兑换码池。0 未使用 1 已发放 2 作废';

CREATE TABLE IF NOT EXISTS wys_mall_cart_item (
  user_id    varchar(64) NOT NULL,
  sku_id     bigint NOT NULL,
  qty        integer NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, sku_id),
  CONSTRAINT chk_wys_mall_cart_qty CHECK (qty > 0)
);
COMMENT ON TABLE wys_mall_cart_item IS '购物车。一人一 SKU 一行';

CREATE TABLE IF NOT EXISTS wys_mall_order (
  order_id         bigserial PRIMARY KEY,
  order_no         varchar(32) NOT NULL,
  store_id         integer NOT NULL,
  buyer_user_id    varchar(64) NOT NULL,
  idempotency_key  varchar(64) NOT NULL,
  status           smallint NOT NULL DEFAULT 0,
  payment_channel  smallint,
  amount           numeric(10,2) NOT NULL,
  receiver_name    text,
  receiver_phone   text,
  receiver_address text,
  paid_at          timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_mall_order_status CHECK (status IN (0, 1, 2, 3, 4)),
  CONSTRAINT chk_wys_mall_order_channel CHECK (payment_channel IS NULL OR payment_channel IN (1, 2, 3, 4)),
  CONSTRAINT chk_wys_mall_order_amount CHECK (amount >= 0),
  CONSTRAINT chk_wys_mall_order_channel_when_paid CHECK (
    (status IN (0, 3) AND payment_channel IS NULL)
    OR (status IN (1, 2, 4) AND payment_channel IS NOT NULL)
  )
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_order_no ON wys_mall_order (order_no);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_order_idem ON wys_mall_order (buyer_user_id, idempotency_key);
CREATE INDEX IF NOT EXISTS ix_wys_mall_order_buyer ON wys_mall_order (buyer_user_id, created_at DESC);
COMMENT ON TABLE wys_mall_order IS '订单。payment_channel 1 支付宝 2 微信 3 苹果内购 4 华为内购';
COMMENT ON COLUMN wys_mall_order.payment_channel IS '1=支付宝 2=微信 3=苹果内购 4=华为内购；未支付为空';
COMMENT ON COLUMN wys_mall_order.status IS '0=待支付 1=已支付 2=已履约 3=已取消 4=已关闭';

CREATE TABLE IF NOT EXISTS wys_mall_order_item (
  item_id       bigserial PRIMARY KEY,
  order_id      bigint NOT NULL,
  sku_id        bigint NOT NULL,
  product_id    bigint NOT NULL,
  kind          smallint NOT NULL,
  product_title text NOT NULL,
  cover_url     text,
  specs         jsonb NOT NULL DEFAULT '{}',
  price         numeric(10,2) NOT NULL,
  qty           integer NOT NULL,
  line_amount   numeric(10,2) NOT NULL,
  content_url   text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_mall_order_item_qty CHECK (qty > 0),
  CONSTRAINT chk_wys_mall_order_item_kind CHECK (kind IN (0, 1))
);
CREATE INDEX IF NOT EXISTS ix_wys_mall_order_item_order ON wys_mall_order_item (order_id);
CREATE INDEX IF NOT EXISTS ix_wys_mall_order_item_sku ON wys_mall_order_item (sku_id);
COMMENT ON TABLE wys_mall_order_item IS '订单行快照。商品名、图、规格、价格冗余存储';

CREATE TABLE IF NOT EXISTS wys_mall_order_log (
  log_id        bigserial PRIMARY KEY,
  order_id      bigint NOT NULL,
  from_status   smallint,
  to_status     smallint NOT NULL,
  actor_user_id varchar(64) NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_wys_mall_order_log_order ON wys_mall_order_log (order_id);
COMMENT ON TABLE wys_mall_order_log IS '订单状态变更日志，只追加';

CREATE TABLE IF NOT EXISTS wys_mall_payment (
  payment_id       bigserial PRIMARY KEY,
  payment_no       varchar(32) NOT NULL,
  order_id         bigint NOT NULL,
  user_id          varchar(64) NOT NULL,
  payment_channel  smallint NOT NULL,
  amount           numeric(10,2) NOT NULL,
  status           smallint NOT NULL DEFAULT 0,
  channel_trade_no varchar(128),
  channel_payload  jsonb NOT NULL DEFAULT '{}',
  paid_at          timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_mall_payment_channel CHECK (payment_channel IN (1, 2, 3, 4)),
  CONSTRAINT chk_wys_mall_payment_status CHECK (status IN (0, 1, 2)),
  CONSTRAINT chk_wys_mall_payment_amount CHECK (amount >= 0)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_payment_no ON wys_mall_payment (payment_no);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_payment_trade
  ON wys_mall_payment (payment_channel, channel_trade_no) WHERE channel_trade_no IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_payment_one_success
  ON wys_mall_payment (order_id) WHERE status = 1;
CREATE INDEX IF NOT EXISTS ix_wys_mall_payment_order ON wys_mall_payment (order_id);
CREATE INDEX IF NOT EXISTS ix_wys_mall_payment_user ON wys_mall_payment (user_id);
CREATE INDEX IF NOT EXISTS ix_wys_mall_payment_channel ON wys_mall_payment (payment_channel, status);
COMMENT ON TABLE wys_mall_payment IS '支付单。本地模拟支付任意渠道均可直接成功';
COMMENT ON COLUMN wys_mall_payment.payment_channel IS '1=支付宝 2=微信 3=苹果内购 4=华为内购';
COMMENT ON COLUMN wys_mall_payment.status IS '0=处理中 1=成功 2=失败';

CREATE TABLE IF NOT EXISTS wys_mall_refund (
  refund_id         bigserial PRIMARY KEY,
  refund_no         varchar(32) NOT NULL,
  payment_id        bigint NOT NULL,
  order_id          bigint NOT NULL,
  user_id           varchar(64) NOT NULL,
  payment_channel   smallint NOT NULL,
  amount            numeric(10,2) NOT NULL,
  status            smallint NOT NULL DEFAULT 0,
  channel_refund_no varchar(128),
  channel_payload   jsonb NOT NULL DEFAULT '{}',
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_mall_refund_channel CHECK (payment_channel IN (1, 2, 3, 4)),
  CONSTRAINT chk_wys_mall_refund_status CHECK (status IN (0, 1, 2)),
  CONSTRAINT chk_wys_mall_refund_amount CHECK (amount > 0)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_refund_no ON wys_mall_refund (refund_no);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_refund_trade
  ON wys_mall_refund (payment_channel, channel_refund_no) WHERE channel_refund_no IS NOT NULL;
CREATE INDEX IF NOT EXISTS ix_wys_mall_refund_order ON wys_mall_refund (order_id);
CREATE INDEX IF NOT EXISTS ix_wys_mall_refund_payment ON wys_mall_refund (payment_id);
COMMENT ON TABLE wys_mall_refund IS '退款单。payment_channel 与原支付相同';
COMMENT ON COLUMN wys_mall_refund.payment_channel IS '1=支付宝 2=微信 3=苹果内购 4=华为内购';

CREATE TABLE IF NOT EXISTS wys_mall_audit_log (
  audit_id      bigserial PRIMARY KEY,
  actor_user_id varchar(64) NOT NULL,
  action        varchar(64) NOT NULL,
  target_type   varchar(32) NOT NULL,
  target_id     varchar(64) NOT NULL,
  detail        jsonb NOT NULL DEFAULT '{}',
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_wys_mall_audit_target ON wys_mall_audit_log (target_type, target_id);
COMMENT ON TABLE wys_mall_audit_log IS '商城审计日志。不落敏感明文';

INSERT INTO permission (code, name) VALUES
  ('mall.catalog.write', '写本店商城目录')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permission (role_code, permission_code) VALUES
  ('store_admin', 'mall.catalog.write'),
  ('store_staff', 'mall.catalog.write'),
  ('platform_admin', 'mall.catalog.write')
ON CONFLICT (role_code, permission_code) DO NOTHING;

-- 积分价 / 混合支付（可重复执行）
ALTER TABLE wys_mall_sku ADD COLUMN IF NOT EXISTS price_points bigint NOT NULL DEFAULT 0;
ALTER TABLE wys_mall_order ADD COLUMN IF NOT EXISTS total_points bigint NOT NULL DEFAULT 0;
ALTER TABLE wys_mall_order ADD COLUMN IF NOT EXISTS payment_mode smallint NOT NULL DEFAULT 1;
ALTER TABLE wys_mall_order_item ADD COLUMN IF NOT EXISTS price_points bigint NOT NULL DEFAULT 0;
ALTER TABLE wys_mall_order_item ADD COLUMN IF NOT EXISTS line_points bigint NOT NULL DEFAULT 0;
ALTER TABLE wys_mall_payment DROP CONSTRAINT IF EXISTS chk_wys_mall_payment_channel;
ALTER TABLE wys_mall_order DROP CONSTRAINT IF EXISTS chk_wys_mall_order_channel;
ALTER TABLE wys_mall_order ADD CONSTRAINT chk_wys_mall_order_channel
  CHECK (payment_channel IS NULL OR payment_channel IN (1, 2, 3, 4, 5, 6));
ALTER TABLE wys_mall_payment ADD CONSTRAINT chk_wys_mall_payment_channel
  CHECK (payment_channel IN (1, 2, 3, 4, 5, 6));
DROP INDEX IF EXISTS uq_wys_mall_payment_one_success;
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_mall_payment_one_success_per_channel
  ON wys_mall_payment (order_id, payment_channel) WHERE status = 1;
