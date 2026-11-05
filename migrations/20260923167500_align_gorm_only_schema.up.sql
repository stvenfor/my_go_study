-- 补齐「只有 GORM AutoMigrate 建过、migrations/ 一直没有」的 10 张表。
--
-- 来源：dev 库（AutoMigrate + 内嵌 SQL 实际跑出来的形态）逐表 pg_dump --schema-only 对齐。
-- 目的（parity-p0-1）：关掉 database.auto_migrate 后，`make migrate-up` 出来的库
-- 就是应用运行需要的完整 schema，正式环境启动不再碰 DDL。
-- 全部语句幂等（IF NOT EXISTS），已有这些表的老库执行本文件等于 no-op。

-- 本文件里两张表带 trigram 索引，扩展必须早于索引存在（20260923170000 也建一次，幂等）。
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
  id          bigserial PRIMARY KEY,
  token_hash  character varying(64) NOT NULL,
  expires_at  timestamp with time zone NOT NULL,
  created_at  timestamp with time zone,
  user_id     character varying(64) REFERENCES users (user_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_refresh_tokens_token_hash ON auth_refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_expires_at ON auth_refresh_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_user_id ON auth_refresh_tokens (user_id);

-- Casbin 策略表：由 casbin gorm-adapter 首次构造 Enforcer 时建，这里前置收进迁移。
CREATE TABLE IF NOT EXISTS casbin_rule (
  id    bigserial PRIMARY KEY,
  ptype character varying(100),
  v0    character varying(100),
  v1    character varying(100),
  v2    character varying(100),
  v3    character varying(100),
  v4    character varying(100),
  v5    character varying(100)
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_index ON casbin_rule (ptype, v0, v1, v2, v3, v4, v5);

CREATE TABLE IF NOT EXISTS transactions (
  id         bigserial PRIMARY KEY,
  user_id    character varying(64) REFERENCES users (user_id),
  type       character varying(32) NOT NULL,
  category   character varying(64) NOT NULL,
  amount     numeric NOT NULL,
  date       character varying(32) NOT NULL,
  note       text,
  created_at timestamp with time zone,
  updated_at timestamp with time zone
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions (user_id);

CREATE TABLE IF NOT EXISTS transaction_records (
  id         bigserial PRIMARY KEY,
  user_id    bigint NOT NULL,
  type       character varying(32) NOT NULL,
  category   character varying(128) NOT NULL,
  amount     numeric NOT NULL,
  date       character varying(32) NOT NULL,
  note       character varying(512),
  created_at timestamp with time zone,
  updated_at timestamp with time zone
);
CREATE INDEX IF NOT EXISTS idx_transaction_records_user_id ON transaction_records (user_id);

CREATE TABLE IF NOT EXISTS wys_topics (
  id               uuid PRIMARY KEY,
  name             character varying(200) NOT NULL,
  heat             bigint DEFAULT 0 NOT NULL,
  is_ask_everyone  boolean DEFAULT false NOT NULL,
  created_at       timestamp with time zone
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_wys_topics_name ON wys_topics (name);
CREATE INDEX IF NOT EXISTS idx_wys_topics_name_trgm ON wys_topics USING gin (name gin_trgm_ops);

CREATE TABLE IF NOT EXISTS wys_posts (
  id              uuid PRIMARY KEY,
  user_id         character varying(64) NOT NULL,
  content         text NOT NULL,
  media_type      smallint DEFAULT 0 NOT NULL,
  image_urls      jsonb DEFAULT '[]'::jsonb NOT NULL,
  video_url       text,
  video_cover_url text,
  topic_id        uuid,
  is_ask_everyone boolean DEFAULT false NOT NULL,
  source          character varying(64) DEFAULT ''::character varying NOT NULL,
  like_count      bigint DEFAULT 0 NOT NULL,
  comment_count   bigint DEFAULT 0 NOT NULL,
  heat            bigint DEFAULT 0 NOT NULL,
  deleted_at      timestamp with time zone,
  created_at      timestamp with time zone,
  updated_at      timestamp with time zone
);
CREATE INDEX IF NOT EXISTS idx_wys_posts_user_id ON wys_posts (user_id);
CREATE INDEX IF NOT EXISTS idx_wys_posts_topic_id ON wys_posts (topic_id);
CREATE INDEX IF NOT EXISTS idx_wys_posts_created_at ON wys_posts (created_at);
CREATE INDEX IF NOT EXISTS idx_wys_posts_deleted_at ON wys_posts (deleted_at);
CREATE INDEX IF NOT EXISTS idx_wys_posts_heat ON wys_posts (heat);
CREATE INDEX IF NOT EXISTS idx_wys_posts_content_trgm ON wys_posts USING gin (content gin_trgm_ops);

CREATE TABLE IF NOT EXISTS wys_post_comments (
  id                uuid PRIMARY KEY,
  post_id           uuid NOT NULL,
  user_id           character varying(64) NOT NULL,
  content           text NOT NULL,
  reply_to_nickname character varying(100),
  deleted_at        timestamp with time zone,
  created_at        timestamp with time zone
);
CREATE INDEX IF NOT EXISTS idx_wys_post_comments_post_id ON wys_post_comments (post_id);
CREATE INDEX IF NOT EXISTS idx_wys_post_comments_deleted_at ON wys_post_comments (deleted_at);

CREATE TABLE IF NOT EXISTS wys_post_likes (
  post_id    uuid NOT NULL,
  user_id    character varying(64) NOT NULL,
  created_at timestamp with time zone,
  PRIMARY KEY (post_id, user_id)
);

CREATE TABLE IF NOT EXISTS wys_user_follows (
  follower_id character varying(64) NOT NULL,
  followee_id character varying(64) NOT NULL,
  created_at  timestamp with time zone,
  PRIMARY KEY (follower_id, followee_id)
);
CREATE INDEX IF NOT EXISTS idx_wys_user_follows_followee_id ON wys_user_follows (followee_id);

CREATE TABLE IF NOT EXISTS wys_auto_finance_product (
  id                      bigserial PRIMARY KEY,
  name                    character varying(128) NOT NULL,
  annual_rate_bps         bigint NOT NULL,
  allowed_terms_csv       character varying(64) NOT NULL,
  min_down_payment_bps    bigint NOT NULL,
  one_time_fee_fen        bigint DEFAULT 0 NOT NULL,
  subsidy_type            smallint DEFAULT 0 NOT NULL,
  subsidy_value           bigint DEFAULT 0 NOT NULL,
  compulsory_insurance_fen bigint DEFAULT 0 NOT NULL,
  commercial_rate_bps     bigint DEFAULT 0 NOT NULL,
  enabled                 boolean DEFAULT true NOT NULL,
  created_at              timestamp with time zone,
  updated_at              timestamp with time zone
);

-- 以下 4 张此前只由内嵌 SQL（repository/postgres/sql/points_schema.sql）建立。
CREATE TABLE IF NOT EXISTS wys_points_wallet (
  user_id    text PRIMARY KEY,
  balance    bigint DEFAULT 0 NOT NULL,
  updated_at timestamp with time zone DEFAULT now() NOT NULL,
  CONSTRAINT wys_points_wallet_balance_check CHECK (balance >= 0)
);

CREATE TABLE IF NOT EXISTS wys_points_ledger (
  ledger_id  bigserial PRIMARY KEY,
  user_id    text NOT NULL,
  delta      bigint NOT NULL,
  balance    bigint NOT NULL,
  reason     text NOT NULL,
  ref_id     text DEFAULT ''::text NOT NULL,
  created_at timestamp with time zone DEFAULT now() NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_wys_points_ledger_user ON wys_points_ledger (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS wys_check_ins (
  user_id    text NOT NULL,
  day        text NOT NULL,
  streak     integer DEFAULT 1 NOT NULL,
  points     bigint NOT NULL,
  created_at timestamp with time zone DEFAULT now() NOT NULL,
  PRIMARY KEY (user_id, day)
);

CREATE TABLE IF NOT EXISTS wys_task_claims (
  user_id    text NOT NULL,
  day        text NOT NULL,
  task_code  text NOT NULL,
  points     bigint NOT NULL,
  created_at timestamp with time zone DEFAULT now() NOT NULL,
  PRIMARY KEY (user_id, day, task_code)
);
