-- 合并为单表：用户-门店统计（数字字段可直接在库中核对）
-- role: 0=销售顾问 1=销售经理 2=总经理

DROP TABLE IF EXISTS wys_store_customers;
DROP TABLE IF EXISTS wys_store_members;
DROP TABLE IF EXISTS wys_stores;

CREATE TABLE IF NOT EXISTS wys_user_store_stats (
  store_id         text NOT NULL,
  user_id          uuid NOT NULL,
  store_name       text NOT NULL DEFAULT '',
  role             smallint NOT NULL DEFAULT 0,
  days_joined      integer NOT NULL DEFAULT 0,
  employee_count   integer NOT NULL DEFAULT 0,
  store_days       integer NOT NULL DEFAULT 0,
  total_customers  integer NOT NULL DEFAULT 0,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (store_id, user_id),
  CONSTRAINT chk_wys_user_store_stats_role CHECK (role IN (0, 1, 2)),
  CONSTRAINT chk_wys_user_store_stats_nonneg CHECK (
    days_joined >= 0
    AND employee_count >= 0
    AND store_days >= 0
    AND total_customers >= 0
  )
);

COMMENT ON TABLE wys_user_store_stats IS 'Mine 用户门店统计（wys_ 前缀单表）';
COMMENT ON COLUMN wys_user_store_stats.role IS '0=销售顾问 1=销售经理 2=总经理';
COMMENT ON COLUMN wys_user_store_stats.days_joined IS '加入天数（整数，库内直接可见）';
COMMENT ON COLUMN wys_user_store_stats.employee_count IS '员工数';
COMMENT ON COLUMN wys_user_store_stats.store_days IS '店铺天数';
COMMENT ON COLUMN wys_user_store_stats.total_customers IS '累计客户';

CREATE INDEX IF NOT EXISTS idx_wys_user_store_stats_user_id
  ON wys_user_store_stats (user_id);
