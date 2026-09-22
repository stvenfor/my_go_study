-- 小视频：独立实体 + 点赞
CREATE TABLE IF NOT EXISTS wys_short_videos (
  id uuid PRIMARY KEY,
  user_id varchar(64) NOT NULL,
  title text NOT NULL,
  video_url text NOT NULL,
  cover_url text NOT NULL,
  duration varchar(32) NOT NULL DEFAULT '0:15',
  aspect_ratio double precision NOT NULL DEFAULT 1.25,
  topic_id uuid,
  status smallint NOT NULL DEFAULT 0,
  view_count bigint NOT NULL DEFAULT 0,
  like_count integer NOT NULL DEFAULT 0,
  approved_at timestamptz,
  deleted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_short_videos_status CHECK (status IN (0, 1))
);
CREATE INDEX IF NOT EXISTS idx_wys_short_videos_discovery
  ON wys_short_videos (deleted_at, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wys_short_videos_user
  ON wys_short_videos (user_id, deleted_at, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wys_short_videos_topic
  ON wys_short_videos (topic_id);
COMMENT ON TABLE wys_short_videos IS '小视频；status 0=reviewing 1=normal';

CREATE TABLE IF NOT EXISTS wys_short_video_likes (
  short_video_id uuid NOT NULL,
  user_id varchar(64) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (short_video_id, user_id)
);
COMMENT ON TABLE wys_short_video_likes IS '小视频点赞';
