-- 售后专区：维修保养记录（与 migrations/20260923090000_after_sales_zone.up.sql 对齐）
CREATE TABLE IF NOT EXISTS wys_after_sales_record (
  record_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  appointment_id bigint,
  customer_id bigint,
  customer_user_id varchar(64),
  customer_name varchar(128) NOT NULL DEFAULT '',
  customer_phone varchar(32) NOT NULL DEFAULT '',
  plate_no varchar(32) NOT NULL DEFAULT '',
  mileage integer,
  service_kind smallint NOT NULL,
  title varchar(256) NOT NULL,
  content text NOT NULL DEFAULT '',
  service_date date NOT NULL,
  created_by varchar(64) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_after_sales_record_kind CHECK (service_kind IN (0, 1))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_wys_after_sales_record_appointment
  ON wys_after_sales_record (appointment_id)
  WHERE appointment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wys_after_sales_record_store_created
  ON wys_after_sales_record (store_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_wys_after_sales_record_customer_user
  ON wys_after_sales_record (customer_user_id, created_at DESC)
  WHERE customer_user_id IS NOT NULL;

COMMENT ON TABLE wys_after_sales_record IS '售后专区维修保养记录。service_kind 0=repair 1=maintenance；appointment_id 可选且唯一';
