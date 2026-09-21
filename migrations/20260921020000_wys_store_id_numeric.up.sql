-- store_id 改为唯一整数主键，并记录用户当前店铺
DROP TABLE IF EXISTS wys_user_store_stats;

CREATE TABLE wys_user_store_stats (
  store_id         integer PRIMARY KEY,
  user_id          varchar(64) NOT NULL,
  store_name       text NOT NULL DEFAULT '',
  role             smallint NOT NULL DEFAULT 0,
  days_joined      integer NOT NULL DEFAULT 0,
  employee_count   integer NOT NULL DEFAULT 0,
  store_days       integer NOT NULL DEFAULT 0,
  total_customers  integer NOT NULL DEFAULT 0,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_user_store_stats_store_id CHECK (store_id > 0),
  CONSTRAINT chk_wys_user_store_stats_role CHECK (role IN (0, 1, 2)),
  CONSTRAINT chk_wys_user_store_stats_nonneg CHECK (
    days_joined >= 0
    AND employee_count >= 0
    AND store_days >= 0
    AND total_customers >= 0
  )
);

COMMENT ON TABLE wys_user_store_stats IS 'Mine 用户门店统计；store_id 全局唯一数字';
COMMENT ON COLUMN wys_user_store_stats.store_id IS '店铺 ID，唯一正整数';
COMMENT ON COLUMN wys_user_store_stats.role IS '0=销售顾问 1=销售经理 2=总经理';
COMMENT ON COLUMN wys_user_store_stats.days_joined IS '加入天数';
COMMENT ON COLUMN wys_user_store_stats.employee_count IS '员工数';
COMMENT ON COLUMN wys_user_store_stats.store_days IS '店铺天数';
COMMENT ON COLUMN wys_user_store_stats.total_customers IS '累计客户';

CREATE INDEX IF NOT EXISTS idx_wys_user_store_stats_user_id
  ON wys_user_store_stats (user_id);

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS current_store_id integer;

COMMENT ON COLUMN users.current_store_id IS '当前选中店铺，对应 wys_user_store_stats.store_id';
