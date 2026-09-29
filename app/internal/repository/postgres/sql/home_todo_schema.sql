-- 首页待办：入店申请、跟进客户、售后预约、店务审核单
CREATE TABLE IF NOT EXISTS wys_store_join_application (
  application_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  applicant_user_id varchar(64) NOT NULL,
  status smallint NOT NULL DEFAULT 0,
  reviewed_by varchar(64),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_store_join_application_status CHECK (status IN (0, 1, 2))
);
CREATE INDEX IF NOT EXISTS idx_wys_store_join_app_store_status
  ON wys_store_join_application (store_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_store_join_app_pending
  ON wys_store_join_application (store_id, applicant_user_id)
  WHERE status = 0;
COMMENT ON TABLE wys_store_join_application IS '入店申请。0=pending 1=approved 2=rejected';

CREATE TABLE IF NOT EXISTS wys_store_customer (
  customer_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  display_name varchar(128) NOT NULL DEFAULT '',
  next_follow_up_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wys_store_customer_follow
  ON wys_store_customer (store_id, next_follow_up_at);
COMMENT ON TABLE wys_store_customer IS '门店客户；next_follow_up_at 到期计待跟进';

CREATE TABLE IF NOT EXISTS wys_after_sales_appointment (
  appointment_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  customer_name varchar(128) NOT NULL DEFAULT '',
  appointment_date date NOT NULL,
  status smallint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_after_sales_appointment_status CHECK (status IN (0, 1))
);
CREATE INDEX IF NOT EXISTS idx_wys_after_sales_store_status_date
  ON wys_after_sales_appointment (store_id, status, appointment_date);
COMMENT ON TABLE wys_after_sales_appointment IS '售后预约。0=pending 1=done；首页 count=pending 且预约日>=今天';

CREATE TABLE IF NOT EXISTS wys_store_review_order (
  order_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  title varchar(256) NOT NULL DEFAULT '',
  status smallint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_store_review_order_status CHECK (status IN (0, 1, 2))
);
CREATE INDEX IF NOT EXISTS idx_wys_store_review_order_store_status
  ON wys_store_review_order (store_id, status);
COMMENT ON TABLE wys_store_review_order IS '店务审核单（非商城）。0=pending 1=approved 2=rejected';

CREATE TABLE IF NOT EXISTS wys_home_todo_packing_demo (
  store_id integer PRIMARY KEY,
  large_n integer NOT NULL DEFAULT 1,
  medium_n integer NOT NULL DEFAULT 3,
  small_n integer NOT NULL DEFAULT 4,
  created_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE wys_home_todo_packing_demo IS '首页待办装箱演示规格：大/中/小卡目标张数';
