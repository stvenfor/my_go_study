-- =============================================================================
-- 本地 Postgres 建表脚本（Navicat Premium 可直接执行）
-- 库：my_go_study | 对齐 auth.provider=local + GORM AutoMigrate
-- 连接：127.0.0.1:5432 / postgres / postgres / db=my_go_study
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- 1. 本地登录用户（对齐 Supabase auth.users 的 UUID 身份）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth_users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email         varchar(255) NOT NULL,
  phone         varchar(32),
  password_hash varchar(255) NOT NULL,
  display_name  varchar(128),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_users_email ON auth_users (email);
CREATE INDEX IF NOT EXISTS idx_auth_users_phone ON auth_users (phone);

-- ---------------------------------------------------------------------------
-- 2. 本地 refresh token
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
  id         bigserial PRIMARY KEY,
  user_id    uuid NOT NULL,
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
-- 3. 用户资料
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS profiles (
  id           uuid PRIMARY KEY,
  display_name text,
  avatar_url   text,
  phone        text,
  created_at   timestamptz DEFAULT now(),
  updated_at   timestamptz DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_phone ON profiles (phone)
  WHERE phone IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 4. 收支 / 二手车列表示例（Flutter GET /api/v1/transactions）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS transactions (
  id         bigserial PRIMARY KEY,
  user_id    uuid,
  type       varchar(32) NOT NULL,
  category   varchar(64) NOT NULL,
  amount     double precision NOT NULL,
  date       varchar(32) NOT NULL,  -- YYYY-MM-DD
  note       text,
  created_at timestamptz,
  updated_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions (user_id);

-- ---------------------------------------------------------------------------
-- （可选）遗留表 —— Flutter 主路径不依赖，可按需执行
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
  id         bigserial PRIMARY KEY,
  username   varchar(64) NOT NULL,
  email      varchar(128) NOT NULL,
  password   varchar(255) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

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
