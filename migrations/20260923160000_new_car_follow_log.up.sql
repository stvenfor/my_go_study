-- 新车跟进流水
CREATE TABLE IF NOT EXISTS wys_new_car_follow_log (
  log_id bigserial PRIMARY KEY,
  file_id bigint NOT NULL REFERENCES wys_new_car_follow_file(file_id) ON DELETE CASCADE,
  author_user_id varchar(64) NOT NULL,
  body varchar(2000) NOT NULL,
  follow_level char(1) NOT NULL DEFAULT '',
  next_follow_up_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_new_car_follow_log_level CHECK (follow_level = '' OR follow_level IN ('A', 'B', 'E', 'H'))
);

CREATE INDEX IF NOT EXISTS idx_wys_new_car_follow_log_file_created
  ON wys_new_car_follow_log (file_id, created_at DESC);

COMMENT ON TABLE wys_new_car_follow_log IS '新车跟进流水；可选带 follow_level / next_follow_up_at';
