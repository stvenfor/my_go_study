-- 尽力回滚到 UUID id + display_name + account_id。
-- 不恢复已丢失的锁定次数以外的业务含义；user_id 必须是合法 UUID。

ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_status;
DROP INDEX IF EXISTS idx_users_email_open;
DROP INDEX IF EXISTS idx_users_phone_open;

ALTER TABLE users ADD COLUMN IF NOT EXISTS id uuid;
UPDATE users SET id = user_id::uuid WHERE id IS NULL AND user_id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';

ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name varchar(128);
UPDATE users SET display_name = user_name WHERE display_name IS NULL;

DO $$
BEGIN
  IF to_regclass('auth_refresh_tokens') IS NOT NULL THEN
    ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_user_id_fkey;
    ALTER TABLE auth_refresh_tokens ADD COLUMN IF NOT EXISTS account_id uuid;
    UPDATE auth_refresh_tokens t
    SET account_id = u.id
    FROM users u
    WHERE t.user_id = u.user_id AND t.account_id IS NULL;
    ALTER TABLE auth_refresh_tokens DROP COLUMN IF EXISTS user_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('transactions') IS NOT NULL THEN
    ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_user_id_fkey;
  END IF;
END $$;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE users DROP COLUMN IF EXISTS user_name;
ALTER TABLE users DROP COLUMN IF EXISTS status;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS phone_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS failed_login_count;
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS password_changed_at;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE users ALTER COLUMN id SET NOT NULL;
ALTER TABLE users ADD PRIMARY KEY (id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_user_id ON users (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

DO $$
BEGIN
  IF to_regclass('auth_refresh_tokens') IS NOT NULL THEN
    ALTER TABLE auth_refresh_tokens
      ADD CONSTRAINT auth_refresh_tokens_account_id_fkey
      FOREIGN KEY (account_id) REFERENCES users (id) ON DELETE CASCADE;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('transactions') IS NOT NULL THEN
    ALTER TABLE transactions
      ADD CONSTRAINT transactions_user_id_fkey
      FOREIGN KEY (user_id) REFERENCES users (user_id);
  END IF;
END $$;
