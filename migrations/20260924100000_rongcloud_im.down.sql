-- 回滚融云 IM 表（与 20260924100000_rongcloud_im.up.sql 对应）。
DROP TABLE IF EXISTS wys_im_message_backup;
DROP TABLE IF EXISTS wys_im_store_group;
DROP TABLE IF EXISTS wys_im_free_group_member;
DROP TABLE IF EXISTS wys_im_free_group;
DROP TABLE IF EXISTS wys_im_friendship;
DROP TABLE IF EXISTS wys_im_friend_request;
