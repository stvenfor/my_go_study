-- IM: friends, groups, message backup
CREATE TABLE IF NOT EXISTS wys_im_friend_request (
  id           text PRIMARY KEY,
  from_user_id text NOT NULL,
  to_user_id   text NOT NULL,
  status       text NOT NULL DEFAULT 'pending',
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_im_friend_request_status CHECK (status IN ('pending', 'accepted', 'rejected'))
);
CREATE INDEX IF NOT EXISTS ix_wys_im_friend_request_to ON wys_im_friend_request (to_user_id, status);
CREATE INDEX IF NOT EXISTS ix_wys_im_friend_request_from ON wys_im_friend_request (from_user_id, status);

CREATE TABLE IF NOT EXISTS wys_im_friendship (
  user_a     text NOT NULL,
  user_b     text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_a, user_b),
  CONSTRAINT chk_wys_im_friendship_order CHECK (user_a < user_b)
);

CREATE TABLE IF NOT EXISTS wys_im_free_group (
  group_id      text PRIMARY KEY,
  name          text NOT NULL,
  owner_user_id text NOT NULL,
  dismissed_at  timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ix_wys_im_free_group_owner ON wys_im_free_group (owner_user_id);

CREATE TABLE IF NOT EXISTS wys_im_free_group_member (
  group_id  text NOT NULL,
  user_id   text NOT NULL,
  joined_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, user_id)
);

CREATE TABLE IF NOT EXISTS wys_im_store_group (
  store_id   text PRIMARY KEY,
  group_id   text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wys_im_message_backup (
  id                bigserial PRIMARY KEY,
  message_uid       text NOT NULL UNIQUE,
  user_id           text NOT NULL,
  direction         text NOT NULL,
  conversation_type text NOT NULL DEFAULT '',
  target_id         text NOT NULL DEFAULT '',
  message_type      text NOT NULL DEFAULT '',
  payload           text NOT NULL DEFAULT '',
  sent_at           timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_im_message_backup_dir CHECK (direction IN ('out', 'in', 'recall'))
);
CREATE INDEX IF NOT EXISTS ix_wys_im_message_backup_user ON wys_im_message_backup (user_id, created_at DESC);
