-- 把 users 收成 user_id 主键，并补上账号状态列。可重复执行。

ALTER TABLE users ADD COLUMN IF NOT EXISTS user_name varchar(128);
ALTER TABLE users ADD COLUMN IF NOT EXISTS status smallint NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_count integer NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at timestamptz;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'users'
      AND column_name = 'display_name'
  ) THEN
    UPDATE users
    SET user_name = COALESCE(NULLIF(btrim(display_name), ''), NULLIF(btrim(user_name), ''), split_part(email, '@', 1), 'user')
    WHERE user_name IS NULL OR btrim(user_name) = '';
    ALTER TABLE users DROP COLUMN display_name;
  END IF;
END $$;

UPDATE users
SET user_name = COALESCE(NULLIF(btrim(user_name), ''), split_part(email, '@', 1), 'user')
WHERE user_name IS NULL OR btrim(user_name) = '';

ALTER TABLE users ALTER COLUMN user_name SET NOT NULL;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'users'
      AND column_name = 'id'
  ) AND EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'users'
      AND column_name = 'user_id'
  ) THEN
    UPDATE users SET user_id = id::text WHERE user_id IS NULL OR btrim(user_id) = '';
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
     ) THEN
    ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_account_id_fkey;
    ALTER TABLE auth_refresh_tokens ADD COLUMN IF NOT EXISTS user_id varchar(64);
    UPDATE auth_refresh_tokens SET user_id = account_id::text WHERE user_id IS NULL OR btrim(user_id) = '';
    ALTER TABLE auth_refresh_tokens DROP COLUMN account_id;
  END IF;
END $$;

-- 先拿掉挂在 users(user_id) 唯一索引上的外键，否则后面删不掉 idx_users_user_id。
DO $$
BEGIN
  IF to_regclass('transactions') IS NOT NULL THEN
    ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_user_id_fkey;
  END IF;
  IF to_regclass('auth_refresh_tokens') IS NOT NULL THEN
    ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_user_id_fkey;
  END IF;
END $$;

DO $$
DECLARE
  pk_name text;
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = CURRENT_SCHEMA()
      AND table_name = 'users'
      AND column_name = 'id'
  ) THEN
    SELECT c.conname INTO pk_name
    FROM pg_constraint c
    JOIN pg_class t ON t.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = t.relnamespace
    WHERE n.nspname = CURRENT_SCHEMA()
      AND t.relname = 'users'
      AND c.contype = 'p';
    IF pk_name IS NOT NULL THEN
      EXECUTE format('ALTER TABLE users DROP CONSTRAINT %I', pk_name);
    END IF;
    ALTER TABLE users DROP COLUMN id;
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_index i
    JOIN pg_class c ON c.oid = i.indrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = CURRENT_SCHEMA()
      AND c.relname = 'users'
      AND i.indisprimary
  ) THEN
    ALTER TABLE users ADD PRIMARY KEY (user_id);
  END IF;
END $$;

DROP INDEX IF EXISTS idx_users_user_id;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_phone;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_open
  ON users (email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_open
  ON users (phone) WHERE deleted_at IS NULL AND phone IS NOT NULL AND btrim(phone) <> '';

ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_status;
ALTER TABLE users ADD CONSTRAINT chk_users_status CHECK (status IN (0, 1, 2));

DO $$
BEGIN
  IF to_regclass('auth_refresh_tokens') IS NOT NULL
     AND EXISTS (
       SELECT 1 FROM information_schema.columns
       WHERE table_schema = CURRENT_SCHEMA()
         AND table_name = 'auth_refresh_tokens'
         AND column_name = 'user_id'
     ) THEN
    ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_user_id_fkey;
    ALTER TABLE auth_refresh_tokens
      ADD CONSTRAINT auth_refresh_tokens_user_id_fkey
      FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE;
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
