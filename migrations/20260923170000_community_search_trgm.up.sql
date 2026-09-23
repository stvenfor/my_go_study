-- community search: trigram indexes for ILIKE contains
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_wys_topics_name_trgm ON wys_topics USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_wys_posts_content_trgm ON wys_posts USING gin (content gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_user_name_trgm ON users USING gin (user_name gin_trgm_ops);
