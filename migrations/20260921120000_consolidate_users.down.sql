-- 尽力回滚：从 users 还原 auth_users / profiles，并把 account_id 改回 user_id。
-- 不恢复已删除的 BIGSERIAL users 行（留在 users_legacy_uint，若存在）。

CREATE TABLE IF NOT EXISTS auth_users (
  id            uuid PRIMARY KEY,
  email         varchar(255) NOT NULL,
  phone         varchar(32),
  password_hash varchar(255) NOT NULL,
  display_name  varchar(128),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS profiles (
  id           uuid PRIMARY KEY,
  display_name text,
  avatar_url   text,
  phone        text,
  created_at   timestamptz DEFAULT now(),
  updated_at   timestamptz DEFAULT now()
);

INSERT INTO auth_users (id, email, phone, password_hash, display_name, created_at, updated_at)
SELECT id, email, NULLIF(phone, ''), password_hash, NULLIF(display_name, ''), created_at, updated_at
FROM users
ON CONFLICT (id) DO NOTHING;

INSERT INTO profiles (id, display_name, avatar_url, phone, created_at, updated_at)
SELECT id, NULLIF(display_name, ''), NULLIF(avatar_url, ''), NULLIF(phone, ''), created_at, updated_at
FROM users
ON CONFLICT (id) DO NOTHING;

ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_user_id_fkey;
ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_account_id_fkey;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'auth_refresh_tokens'
      AND column_name = 'account_id'
  ) THEN
    ALTER TABLE auth_refresh_tokens RENAME COLUMN account_id TO user_id;
  END IF;
END $$;
