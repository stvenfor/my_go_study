CREATE TABLE IF NOT EXISTS wys_points_wallet (
  user_id    text PRIMARY KEY,
  balance    bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wys_points_ledger (
  ledger_id  bigserial PRIMARY KEY,
  user_id    text NOT NULL,
  delta      bigint NOT NULL,
  balance    bigint NOT NULL,
  reason     text NOT NULL,
  ref_id     text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_wys_points_ledger_user
  ON wys_points_ledger (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS wys_check_ins (
  user_id    text NOT NULL,
  day        text NOT NULL,
  streak     integer NOT NULL DEFAULT 1,
  points     bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, day)
);

CREATE TABLE IF NOT EXISTS wys_task_claims (
  user_id    text NOT NULL,
  day        text NOT NULL,
  task_code  text NOT NULL,
  points     bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, day, task_code)
);
