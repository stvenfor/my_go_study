-- 新车跟进档案
CREATE TABLE IF NOT EXISTS wys_new_car_follow_file (
  file_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  owner_user_id varchar(64) NOT NULL,
  customer_id bigint NOT NULL,
  follow_level char(1) NOT NULL,
  stage smallint NOT NULL DEFAULT 0,
  vehicle_interest varchar(256) NOT NULL DEFAULT '',
  budget_note varchar(256) NOT NULL DEFAULT '',
  source varchar(64) NOT NULL DEFAULT '',
  next_follow_up_at timestamptz,
  last_follow_at timestamptz,
  closed_reason varchar(128) NOT NULL DEFAULT '',
  customer_name varchar(128) NOT NULL DEFAULT '',
  customer_phone varchar(32) NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_new_car_follow_level CHECK (follow_level IN ('A', 'B', 'E', 'H')),
  CONSTRAINT chk_wys_new_car_follow_stage CHECK (stage IN (0, 1, 2, 3, 4, 5))
);

CREATE INDEX IF NOT EXISTS idx_wys_new_car_follow_owner_updated
  ON wys_new_car_follow_file (store_id, owner_user_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_wys_new_car_follow_owner_level
  ON wys_new_car_follow_file (store_id, owner_user_id, follow_level);

CREATE INDEX IF NOT EXISTS idx_wys_new_car_follow_owner_next
  ON wys_new_car_follow_file (store_id, owner_user_id, next_follow_up_at);

CREATE INDEX IF NOT EXISTS idx_wys_new_car_follow_customer
  ON wys_new_car_follow_file (store_id, customer_id);

COMMENT ON TABLE wys_new_car_follow_file IS '新车跟进档案；follow_level=A/B/E/H；stage 0新建1跟进中2试驾3已报价4成交5战败';
