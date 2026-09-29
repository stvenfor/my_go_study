-- 合并 auth_users + profiles 为 UUID users，并改用字符串 user_id 关联业务数据。
-- 可重复执行。

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'users'
      AND column_name = 'id'
      AND data_type IN ('bigint', 'integer')
  ) THEN
    ALTER TABLE users RENAME TO users_legacy_uint;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS users (
  id            uuid PRIMARY KEY,
  user_id       varchar(64) NOT NULL,
  email         varchar(255) NOT NULL,
  phone         varchar(32),
  password_hash varchar(255) NOT NULL,
  display_name  varchar(128),
  avatar_url       text,
  current_store_id integer,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_user_id ON users (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_phone ON users (phone);

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'auth_users'
  ) THEN
    INSERT INTO users (
      id, user_id, email, phone, password_hash, display_name, avatar_url, created_at, updated_at
    )
    SELECT
      a.id,
      a.id::text,
      a.email,
      COALESCE(NULLIF(btrim(p.phone), ''), NULLIF(btrim(a.phone), '')),
      a.password_hash,
      COALESCE(NULLIF(btrim(p.display_name), ''), NULLIF(btrim(a.display_name), '')),
      NULLIF(btrim(p.avatar_url), ''),
      COALESCE(a.created_at, now()),
      COALESCE(a.updated_at, now())
    FROM auth_users a
    LEFT JOIN profiles p ON p.id = a.id
    ON CONFLICT (id) DO NOTHING;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'users'
      AND column_name = 'id'
  ) THEN
    UPDATE users SET user_id = id::text WHERE user_id IS NULL OR btrim(user_id) = '';
  END IF;
END $$;

ALTER TABLE users ADD COLUMN IF NOT EXISTS current_store_id integer;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'profiles'
      AND column_name = 'current_store_id'
  ) AND EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'users'
      AND column_name = 'id'
  ) THEN
    UPDATE users u
    SET current_store_id = p.current_store_id
    FROM profiles p
    WHERE u.id = p.id AND p.current_store_id IS NOT NULL;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'transactions'
      AND column_name = 'user_id'
      AND data_type = 'uuid'
  ) THEN
    ALTER TABLE transactions ALTER COLUMN user_id TYPE varchar(64) USING user_id::text;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'auth_refresh_tokens'
      AND column_name = 'user_id'
  ) THEN
    ALTER TABLE auth_refresh_tokens RENAME COLUMN user_id TO account_id;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'wys_user_store_stats'
      AND column_name = 'user_id'
      AND data_type = 'uuid'
  ) THEN
    ALTER TABLE wys_user_store_stats ALTER COLUMN user_id TYPE varchar(64) USING user_id::text;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('auth_refresh_tokens') IS NOT NULL
     AND EXISTS (
       SELECT 1 FROM information_schema.columns
       WHERE table_schema = CURRENT_SCHEMA()
         AND table_name = 'auth_refresh_tokens'
         AND column_name = 'account_id'
     )
     AND EXISTS (
       SELECT 1 FROM information_schema.columns
       WHERE table_schema = CURRENT_SCHEMA()
         AND table_name = 'users'
         AND column_name = 'id'
     ) THEN
    ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_account_id_fkey;
    ALTER TABLE auth_refresh_tokens
      ADD CONSTRAINT auth_refresh_tokens_account_id_fkey
      FOREIGN KEY (account_id) REFERENCES users (id) ON DELETE CASCADE;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('transactions') IS NOT NULL
     AND EXISTS (
       SELECT 1 FROM information_schema.columns
       WHERE table_schema = CURRENT_SCHEMA()
         AND table_name = 'transactions'
         AND column_name = 'user_id'
         AND data_type IN ('character varying', 'text')
     ) THEN
    ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_user_id_fkey;
    ALTER TABLE transactions
      ADD CONSTRAINT transactions_user_id_fkey
      FOREIGN KEY (user_id) REFERENCES users (user_id);
  END IF;
END $$;

DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS auth_users;
