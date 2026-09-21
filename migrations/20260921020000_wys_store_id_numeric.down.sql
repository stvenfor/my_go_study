ALTER TABLE users DROP COLUMN IF EXISTS current_store_id;
ALTER TABLE profiles DROP COLUMN IF EXISTS current_store_id;
DROP TABLE IF EXISTS wys_user_store_stats;
