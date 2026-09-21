-- =============================================================================
-- 本地联调：多用户 + 门店成员 + 角色分配（可重复执行）
-- 所有账号密码明文均为 123456
-- 职务(position) ≠ 权限角色(role)；销售顾问/经理/总经理只写在成员表
-- =============================================================================
-- 账号一览：
--   demo@example.com      平台管理员；店1销售经理、店2总经理
--   manager.wd@example.com  店1店内管理员 + 销售经理
--   advisor.wd@example.com  店1店员 + 销售顾问
--   gm.wd@example.com       店1店内管理员 + 总经理
--   manager.ty@example.com  店2店内管理员 + 销售经理
--   advisor.ty@example.com  店2店员 + 销售顾问
--   disabled@example.com  停用账号（status=1，不能登录业务）
-- =============================================================================

-- bcrypt(123456)
-- $2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.

INSERT INTO users (
  user_id, user_name, email, phone, password_hash, status, failed_login_count, created_at, updated_at
)
VALUES
  ('a1111111-1111-4111-8111-111111111111', '平台演示', 'demo@example.com', NULL,
   '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.', 0, 0, now(), now()),
  ('a2222222-2222-4222-8222-222222222222', '沃德龙经理', 'manager.wd@example.com', '13800000021',
   '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.', 0, 0, now(), now()),
  ('a3333333-3333-4333-8333-333333333333', '沃德龙顾问', 'advisor.wd@example.com', '13800000022',
   '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.', 0, 0, now(), now()),
  ('a4444444-4444-4444-8444-444444444444', '沃德龙总经', 'gm.wd@example.com', '13800000023',
   '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.', 0, 0, now(), now()),
  ('a5555555-5555-4555-8555-555555555555', '腾远经理', 'manager.ty@example.com', '13800000031',
   '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.', 0, 0, now(), now()),
  ('a6666666-6666-4666-8666-666666666666', '腾远顾问', 'advisor.ty@example.com', '13800000032',
   '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.', 0, 0, now(), now()),
  ('a7777777-7777-4777-8777-777777777777', '已停用账号', 'disabled@example.com', NULL,
   '$2a$10$g5CvRfCS2BIzvd4HfyHcO.JzwJPChzXpMM0fWXSmGgisKcnY3B/v.', 1, 0, now(), now())
ON CONFLICT (email) WHERE deleted_at IS NULL DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  user_name = EXCLUDED.user_name,
  phone = EXCLUDED.phone,
  status = EXCLUDED.status,
  failed_login_count = 0,
  locked_until = NULL,
  deleted_at = NULL,
  updated_at = now();

INSERT INTO wys_store (store_id, name)
VALUES
  (1, '[4S]北京沃德龙鼎吉利'),
  (2, '[4S]北京腾远吉利'),
  (3, '[4S]演示空店（尚无成员）')
ON CONFLICT (store_id) DO UPDATE SET
  name = EXCLUDED.name,
  updated_at = now();

-- 成员：职务 0顾问 1经理 2总经理
INSERT INTO wys_store_member (user_id, store_id, position)
SELECT u.user_id, v.store_id, v.position
FROM (
  VALUES
    ('demo@example.com', 1, 1::smallint),
    ('demo@example.com', 2, 2::smallint),
    ('manager.wd@example.com', 1, 1::smallint),
    ('advisor.wd@example.com', 1, 0::smallint),
    ('gm.wd@example.com', 1, 2::smallint),
    ('manager.ty@example.com', 2, 1::smallint),
    ('advisor.ty@example.com', 2, 0::smallint)
) AS v(email, store_id, position)
JOIN users u ON u.email = v.email
ON CONFLICT (user_id, store_id) DO UPDATE SET
  position = EXCLUDED.position,
  updated_at = now();

-- 平台角色
INSERT INTO user_role (user_id, role_code, store_id, granted_by)
SELECT u.user_id, 'platform_admin', NULL, NULL
FROM users u WHERE u.email = 'demo@example.com'
ON CONFLICT (user_id, role_code) WHERE store_id IS NULL DO NOTHING;

-- 店内角色
INSERT INTO user_role (user_id, role_code, store_id, granted_by)
SELECT u.user_id, v.role_code, v.store_id, NULL
FROM (
  VALUES
    ('manager.wd@example.com', 'store_admin', 1),
    ('gm.wd@example.com', 'store_admin', 1),
    ('advisor.wd@example.com', 'store_staff', 1),
    ('manager.ty@example.com', 'store_admin', 2),
    ('advisor.ty@example.com', 'store_staff', 2),
    ('demo@example.com', 'store_admin', 1),
    ('demo@example.com', 'store_admin', 2)
) AS v(email, role_code, store_id)
JOIN users u ON u.email = v.email
ON CONFLICT (user_id, role_code, store_id) WHERE store_id IS NOT NULL DO NOTHING;

-- Mine 统计（每人每店；无行则接口数字为 0）
INSERT INTO wys_user_store_stats (
  store_id, user_id, store_name, role,
  days_joined, employee_count, store_days, total_customers,
  created_at, updated_at
)
SELECT
  v.store_id, u.user_id, s.name, m.position,
  v.days_joined, v.employee_count, v.store_days, v.total_customers,
  now(), now()
FROM (
  VALUES
    ('demo@example.com', 1, 1028, 28, 2059, 9366),
    ('demo@example.com', 2, 300, 12, 900, 400),
    ('manager.wd@example.com', 1, 400, 28, 2059, 2000),
    ('advisor.wd@example.com', 1, 120, 28, 2059, 80),
    ('gm.wd@example.com', 1, 800, 28, 2059, 5000),
    ('manager.ty@example.com', 2, 200, 12, 900, 600),
    ('advisor.ty@example.com', 2, 90, 12, 900, 40)
) AS v(email, store_id, days_joined, employee_count, store_days, total_customers)
JOIN users u ON u.email = v.email
JOIN wys_store s ON s.store_id = v.store_id
JOIN wys_store_member m ON m.user_id = u.user_id AND m.store_id = v.store_id
ON CONFLICT (user_id, store_id) DO UPDATE SET
  store_name = EXCLUDED.store_name,
  role = EXCLUDED.role,
  days_joined = EXCLUDED.days_joined,
  employee_count = EXCLUDED.employee_count,
  store_days = EXCLUDED.store_days,
  total_customers = EXCLUDED.total_customers,
  updated_at = now();

UPDATE users u
SET current_store_id = v.store_id
FROM (
  VALUES
    ('demo@example.com', 1),
    ('manager.wd@example.com', 1),
    ('advisor.wd@example.com', 1),
    ('gm.wd@example.com', 1),
    ('manager.ty@example.com', 2),
    ('advisor.ty@example.com', 2)
) AS v(email, store_id)
WHERE u.email = v.email;
