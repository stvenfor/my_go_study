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
