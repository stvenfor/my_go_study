-- 回退门店组织与权限表。若同一 store_id 已有多行统计，无法恢复「一店一行」主键。

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_current_store_id_fkey;
ALTER TABLE wys_user_store_stats DROP CONSTRAINT IF EXISTS wys_user_store_stats_store_id_fkey;

DROP TABLE IF EXISTS user_role;
DROP TABLE IF EXISTS role_permission;
DROP TABLE IF EXISTS permission;
DROP TABLE IF EXISTS role;
DROP TABLE IF EXISTS wys_store_member;

DO $$
BEGIN
  IF to_regclass('wys_user_store_stats') IS NULL THEN
    RETURN;
  END IF;
  IF EXISTS (
    SELECT 1 FROM wys_user_store_stats GROUP BY store_id HAVING COUNT(*) > 1
  ) THEN
    RAISE EXCEPTION 'cannot restore store_id primary key: duplicate store_id';
  END IF;
  ALTER TABLE wys_user_store_stats DROP CONSTRAINT IF EXISTS wys_user_store_stats_pkey;
  ALTER TABLE wys_user_store_stats ADD PRIMARY KEY (store_id);
END $$;

DROP TABLE IF EXISTS wys_store;
