-- 二手车业务单
CREATE TABLE IF NOT EXISTS wys_used_car_order (
  order_id bigserial PRIMARY KEY,
  store_id integer NOT NULL,
  uploader_user_id varchar(64) NOT NULL,
  kind smallint NOT NULL DEFAULT 0,
  customer_id bigint NOT NULL,
  customer_phone varchar(32) NOT NULL,
  customer_name varchar(128) NOT NULL,
  vehicle_model varchar(128) NOT NULL,
  plate_no varchar(32) NOT NULL,
  vin varchar(32) NOT NULL,
  mileage_km integer NOT NULL,
  model_year integer NOT NULL,
  amount numeric(14,2) NOT NULL,
  status smallint NOT NULL DEFAULT 0,
  image_url text,
  reject_reason text,
  rating_stars smallint,
  submitted_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT wys_used_car_order_status_chk CHECK (status IN (0,1,2,3)),
  CONSTRAINT wys_used_car_order_kind_chk CHECK (kind IN (0,1,2)),
  CONSTRAINT wys_used_car_order_stars_chk CHECK (rating_stars IS NULL OR (rating_stars BETWEEN 1 AND 5))
);

CREATE INDEX IF NOT EXISTS idx_wys_used_car_order_uploader
  ON wys_used_car_order (store_id, uploader_user_id, submitted_at DESC);

CREATE INDEX IF NOT EXISTS idx_wys_used_car_order_store_status
  ON wys_used_car_order (store_id, status);
