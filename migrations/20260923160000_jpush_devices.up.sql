-- +goose Up
CREATE TABLE IF NOT EXISTS wys_push_devices (
  id              bigserial PRIMARY KEY,
  user_id         text NOT NULL,
  device_id       text NOT NULL DEFAULT '',
  platform        text NOT NULL DEFAULT 'unknown',
  registration_id text NOT NULL DEFAULT '',
  alias           text NOT NULL DEFAULT '',
  mock            boolean NOT NULL DEFAULT false,
  updated_at      timestamptz NOT NULL DEFAULT now(),
  created_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_push_devices_platform CHECK (
    platform IN ('ios', 'android', 'harmony', 'ohos', 'unknown')
  )
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_wys_push_devices_user_device
  ON wys_push_devices (user_id, device_id);

CREATE INDEX IF NOT EXISTS ix_wys_push_devices_alias
  ON wys_push_devices (alias) WHERE alias <> '';

CREATE INDEX IF NOT EXISTS ix_wys_push_devices_rid
  ON wys_push_devices (registration_id) WHERE registration_id <> '';
