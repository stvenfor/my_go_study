-- =============================================================================
-- Navicat / psql 种子数据（本地 my_go_study）
-- 演示账号：demo@example.com / 123456
-- 在 Navicat 中：选中 my_go_study → 查询 → 新建查询 → 粘贴执行
-- =============================================================================

-- 1) 演示账号（email 唯一；可重复执行）
INSERT INTO auth_users (id, email, phone, password_hash, display_name, created_at, updated_at)
VALUES (
  'a1111111-1111-4111-8111-111111111111',
  'demo@example.com',
  NULL,
  '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.',  -- 明文: 123456
  'Demo User',
  now(),
  now()
)
ON CONFLICT (email) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  display_name = EXCLUDED.display_name,
  updated_at = now();

-- 2) 同步 profiles
INSERT INTO profiles (id, display_name, avatar_url, phone, created_at, updated_at)
SELECT id, COALESCE(display_name, 'Demo User'), NULL, phone, now(), now()
FROM auth_users
WHERE email = 'demo@example.com'
ON CONFLICT (id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  updated_at = now();

-- 3) 示例交易（按 note 去重，可重复执行）
INSERT INTO transactions (user_id, type, category, amount, date, note, created_at, updated_at)
SELECT u.id, v.type, v.category, v.amount, v.date, v.note, now(), now()
FROM auth_users u
JOIN (
  VALUES
    ('demo@example.com', 'expense', '二手车', 128000.00, '2026-09-01', 'navicat-seed: 捷达'),
    ('demo@example.com', 'income',  '退款',     500.00, '2026-09-05', 'navicat-seed: 定金退款'),
    ('demo@example.com', 'expense', '保养',    1200.00, '2026-09-10', 'navicat-seed: 常规保养')
) AS v(email, type, category, amount, date, note)
  ON u.email = v.email
WHERE NOT EXISTS (
  SELECT 1 FROM transactions t WHERE t.user_id = u.id AND t.note = v.note
);

-- ---------------------------------------------------------------------------
-- Navicat 日常操作备忘
-- ---------------------------------------------------------------------------
-- 改资料：UPDATE profiles SET display_name='新名字' WHERE id='...';
-- 加交易：INSERT INTO transactions (user_id,type,category,amount,date,note,created_at,updated_at)
--         VALUES ('用户uuid','expense','二手车',1000,'2026-09-17','备注',now(),now());
-- 删交易：DELETE FROM transactions WHERE id=...;
-- 改密码：必须先生成 bcrypt，再 UPDATE auth_users SET password_hash='...' WHERE email='...';
--         （不要写明文 123456）
