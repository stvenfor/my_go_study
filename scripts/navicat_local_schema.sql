-- =============================================================================
-- 本地 Postgres 建表脚本（Navicat Premium 可直接执行）
-- 库：my_go_study | 对齐 auth.provider=local + GORM AutoMigrate
-- 连接：127.0.0.1:5432 / postgres / postgres / db=my_go_study
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- 1. 统一用户。user_id 为主键；资料与账号状态在同一行。
-- status: 0=正常 1=停用 2=锁定。email/phone 仅未注销行唯一。
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
  user_id             varchar(64) PRIMARY KEY,
  user_name           varchar(128) NOT NULL,
  email               varchar(255) NOT NULL,
  phone               varchar(32),
  password_hash       varchar(255) NOT NULL,
  status              smallint NOT NULL DEFAULT 0,
  email_verified_at   timestamptz,
  phone_verified_at   timestamptz,
  failed_login_count  integer NOT NULL DEFAULT 0,
  locked_until        timestamptz,
  password_changed_at timestamptz,
  last_login_at       timestamptz,
  avatar_url          text,
  current_store_id    integer,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  deleted_at          timestamptz,
  CONSTRAINT chk_users_status CHECK (status IN (0, 1, 2))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_open
  ON users (email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_open
  ON users (phone) WHERE deleted_at IS NULL AND phone IS NOT NULL AND btrim(phone) <> '';

COMMENT ON COLUMN users.status IS '0=正常 1=停用 2=锁定';
COMMENT ON COLUMN users.deleted_at IS '注销时间；非空则不能登录';

-- ---------------------------------------------------------------------------
-- 2. 本地 refresh token（user_id → users.user_id）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
  id         bigserial PRIMARY KEY,
  user_id    varchar(64) NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
  token_hash varchar(64) NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_refresh_tokens_token_hash
  ON auth_refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_user_id
  ON auth_refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_expires_at
  ON auth_refresh_tokens (expires_at);

-- ---------------------------------------------------------------------------
-- 3. 收支 / 二手车列表示例（user_id = users.user_id）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transactions (
  id         bigserial PRIMARY KEY,
  user_id    varchar(64) REFERENCES users (user_id),
  type       varchar(32) NOT NULL,
  category   varchar(64) NOT NULL,
  amount     double precision NOT NULL,
  date       varchar(32) NOT NULL,
  note       text,
  created_at timestamptz,
  updated_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions (user_id);

-- ---------------------------------------------------------------------------
-- 4. 遗留 uint 交易表（不再作为主路径）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transaction_records (
  id         bigserial PRIMARY KEY,
  user_id    bigint NOT NULL,
  type       varchar(32) NOT NULL,
  category   varchar(128) NOT NULL,
  amount     double precision NOT NULL,
  date       varchar(32) NOT NULL,
  note       varchar(512),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transaction_records_user_id
  ON transaction_records (user_id);

-- ---------------------------------------------------------------------------
-- 5. 门店、成员、权限。统计卡每人每店一行。
-- 已有库以启动 SQL 为准：internal/repository/postgres/sql/access_control.sql
-- position: 0=销售顾问 1=销售经理 2=总经理。统计卡 role 列不再作为职务。
-- ---------------------------------------------------------------------------
DROP TABLE IF EXISTS wys_store_customers;
DROP TABLE IF EXISTS wys_store_members;
DROP TABLE IF EXISTS wys_stores;

CREATE TABLE IF NOT EXISTS wys_store (
  store_id   integer PRIMARY KEY,
  name       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_store_id CHECK (store_id > 0)
);

CREATE TABLE IF NOT EXISTS role (
  code  varchar(64) PRIMARY KEY,
  scope text NOT NULL,
  name  varchar(128) NOT NULL,
  CONSTRAINT chk_role_scope CHECK (scope IN ('platform', 'store'))
);

CREATE TABLE IF NOT EXISTS permission (
  code varchar(64) PRIMARY KEY,
  name varchar(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permission (
  role_code       varchar(64) NOT NULL REFERENCES role (code) ON DELETE RESTRICT,
  permission_code varchar(64) NOT NULL REFERENCES permission (code) ON DELETE RESTRICT,
  PRIMARY KEY (role_code, permission_code)
);

CREATE TABLE IF NOT EXISTS wys_store_member (
  user_id    varchar(64) NOT NULL REFERENCES users (user_id) ON DELETE RESTRICT,
  store_id   integer NOT NULL REFERENCES wys_store (store_id) ON DELETE RESTRICT,
  position   smallint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, store_id),
  CONSTRAINT chk_wys_store_member_position CHECK (position IN (0, 1, 2))
);

CREATE TABLE IF NOT EXISTS user_role (
  user_id    varchar(64) NOT NULL REFERENCES users (user_id) ON DELETE RESTRICT,
  role_code  varchar(64) NOT NULL REFERENCES role (code) ON DELETE RESTRICT,
  store_id   integer NULL REFERENCES wys_store (store_id) ON DELETE RESTRICT,
  granted_by varchar(64) NULL REFERENCES users (user_id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_role_platform
  ON user_role (user_id, role_code) WHERE store_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_role_store
  ON user_role (user_id, role_code, store_id) WHERE store_id IS NOT NULL;

INSERT INTO role (code, scope, name) VALUES
  ('platform_admin', 'platform', '平台管理员'),
  ('store_admin', 'store', '店内管理员'),
  ('store_staff', 'store', '店员')
ON CONFLICT (code) DO NOTHING;

INSERT INTO permission (code, name) VALUES
  ('store.create', '建店'),
  ('member.write', '管理成员'),
  ('role.assign_store', '分配店内角色'),
  ('role.assign_platform', '分配平台角色'),
  ('profile.read', '读资料'),
  ('transaction.read_own', '读自己的收支'),
  ('transaction.write_own', '写自己的收支'),
  ('transaction.read_store', '读本店收支')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permission (role_code, permission_code) VALUES
  ('platform_admin', 'store.create'),
  ('platform_admin', 'member.write'),
  ('platform_admin', 'role.assign_store'),
  ('platform_admin', 'role.assign_platform'),
  ('platform_admin', 'profile.read'),
  ('platform_admin', 'transaction.read_own'),
  ('platform_admin', 'transaction.write_own'),
  ('platform_admin', 'transaction.read_store'),
  ('store_admin', 'member.write'),
  ('store_admin', 'role.assign_store'),
  ('store_admin', 'profile.read'),
  ('store_admin', 'transaction.read_own'),
  ('store_admin', 'transaction.write_own'),
  ('store_admin', 'transaction.read_store'),
  ('store_staff', 'profile.read'),
  ('store_staff', 'transaction.read_own'),
  ('store_staff', 'transaction.write_own')
ON CONFLICT (role_code, permission_code) DO NOTHING;

CREATE TABLE IF NOT EXISTS wys_user_store_stats (
  user_id         varchar(64) NOT NULL,
  store_id        integer NOT NULL REFERENCES wys_store (store_id) ON DELETE RESTRICT,
  store_name      text NOT NULL DEFAULT '',
  role            smallint NOT NULL DEFAULT 0,
  days_joined     integer NOT NULL DEFAULT 0,
  employee_count  integer NOT NULL DEFAULT 0,
  store_days      integer NOT NULL DEFAULT 0,
  total_customers integer NOT NULL DEFAULT 0,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, store_id),
  CONSTRAINT chk_wys_user_store_stats_store_id CHECK (store_id > 0),
  CONSTRAINT chk_wys_user_store_stats_role CHECK (role IN (0, 1, 2)),
  CONSTRAINT chk_wys_user_store_stats_nonneg CHECK (
    days_joined >= 0
    AND employee_count >= 0
    AND store_days >= 0
    AND total_customers >= 0
  )
);

COMMENT ON COLUMN wys_user_store_stats.role IS '遗留职务列，资料接口不读。职务在 wys_store_member.position';

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS current_store_id integer;
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
  product_id  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  store_id    integer NOT NULL,
  category_id bigint,
  kind        smallint NOT NULL,
  title       text NOT NULL,
  cover_url   text,
  status      smallint NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,
  CONSTRAINT chk_wys_mall_product_kind CHECK (kind IN (0, 1)),
  CONSTRAINT chk_wys_mall_product_status CHECK (status IN (0, 1, 2))
);
CREATE INDEX IF NOT EXISTS ix_wys_mall_product_store
  ON wys_mall_product (store_id, status) WHERE deleted_at IS NULL;
COMMENT ON TABLE wys_mall_product IS '商品 SPU。kind 0 实体 1 虚拟';
COMMENT ON COLUMN wys_mall_product.kind IS '0=实体 1=虚拟';
COMMENT ON COLUMN wys_mall_product.status IS '0=草稿 1=在售 2=下架';

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

-- 用户收货地址簿
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

-- mall schema appended from sql/mall_schema.sql
