DROP TABLE IF EXISTS wys_user_store_stats;

-- 回滚到三表结构（与 20260921000000_create_wys_store_stats.up.sql 一致）
CREATE TABLE IF NOT EXISTS wys_stores (
  id         text PRIMARY KEY,
  name       text NOT NULL,
  opened_at  timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wys_store_members (
  store_id   text NOT NULL REFERENCES wys_stores (id) ON DELETE CASCADE,
  user_id    uuid NOT NULL,
  joined_at  timestamptz NOT NULL,
  role       text,
  PRIMARY KEY (store_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_wys_store_members_user_id
  ON wys_store_members (user_id);

CREATE TABLE IF NOT EXISTS wys_store_customers (
  id           bigserial PRIMARY KEY,
  store_id     text NOT NULL REFERENCES wys_stores (id) ON DELETE CASCADE,
  display_name text,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_wys_store_customers_store_id
  ON wys_store_customers (store_id);
