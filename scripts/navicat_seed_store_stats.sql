-- 演示店、统计和成员。不在迁移里做：空库还没有 demo 用户。
-- 职务在成员上。统计卡 role 列只是遗留，接口不读。
-- demo@example.com 同时是平台管理员（本地种子，granted_by 为空）。

INSERT INTO wys_store (store_id, name)
VALUES
  (1, '[4S]北京沃德龙鼎吉利'),
  (2, '[4S]北京腾远吉利')
ON CONFLICT (store_id) DO UPDATE SET
  name = EXCLUDED.name,
  updated_at = now();

INSERT INTO wys_user_store_stats (
  store_id, user_id, store_name, role,
  days_joined, employee_count, store_days, total_customers,
  created_at, updated_at
)
SELECT
  v.store_id,
  u.user_id,
  v.store_name,
  v.role,
  v.days_joined,
  v.employee_count,
  v.store_days,
  v.total_customers,
  now(),
  now()
FROM users u
JOIN (
  VALUES
    (1, '[4S]北京沃德龙鼎吉利', 1::smallint, 1028, 28, 2059, 9366),
    (2, '[4S]北京腾远吉利',     2::smallint,  300, 12,  900,  400)
) AS v(store_id, store_name, role, days_joined, employee_count, store_days, total_customers)
  ON u.email = 'demo@example.com'
ON CONFLICT (user_id, store_id) DO UPDATE SET
  store_name = EXCLUDED.store_name,
  role = EXCLUDED.role,
  days_joined = EXCLUDED.days_joined,
  employee_count = EXCLUDED.employee_count,
  store_days = EXCLUDED.store_days,
  total_customers = EXCLUDED.total_customers,
  updated_at = now();

INSERT INTO wys_store_member (user_id, store_id, position)
SELECT u.user_id, v.store_id, v.position
FROM users u
JOIN (
  VALUES
    (1, 1::smallint),
    (2, 2::smallint)
) AS v(store_id, position)
  ON u.email = 'demo@example.com'
ON CONFLICT (user_id, store_id) DO UPDATE SET
  position = EXCLUDED.position,
  updated_at = now();

INSERT INTO user_role (user_id, role_code, store_id, granted_by)
SELECT u.user_id, 'platform_admin', NULL, NULL
FROM users u
WHERE u.email = 'demo@example.com'
ON CONFLICT (user_id, role_code) WHERE store_id IS NULL DO NOTHING;

UPDATE users
SET current_store_id = 1
WHERE email = 'demo@example.com'
  AND (current_store_id IS NULL OR current_store_id = 0);

-- 测试手机号 OTP：13400000000 → 13400000000@dev.test.local（需已登录过一次）
INSERT INTO wys_user_store_stats (
  store_id, user_id, store_name, role,
  days_joined, employee_count, store_days, total_customers,
  created_at, updated_at
)
SELECT
  v.store_id,
  u.user_id,
  v.store_name,
  v.role,
  v.days_joined,
  v.employee_count,
  v.store_days,
  v.total_customers,
  now(),
  now()
FROM users u
JOIN (
  VALUES
    (1, '[4S]北京沃德龙鼎吉利', 0::smallint, 100, 20, 500, 1000),
    (2, '[4S]北京腾远吉利',     1::smallint,  50, 10, 200,  300)
) AS v(store_id, store_name, role, days_joined, employee_count, store_days, total_customers)
  ON (u.phone = '13400000000' OR u.email = '13400000000@dev.test.local')
 AND u.deleted_at IS NULL
ON CONFLICT (user_id, store_id) DO UPDATE SET
  store_name = EXCLUDED.store_name,
  role = EXCLUDED.role,
  days_joined = EXCLUDED.days_joined,
  employee_count = EXCLUDED.employee_count,
  store_days = EXCLUDED.store_days,
  total_customers = EXCLUDED.total_customers,
  updated_at = now();

INSERT INTO wys_store_member (user_id, store_id, position)
SELECT u.user_id, v.store_id, v.position
FROM users u
JOIN (
  VALUES
    (1, 0::smallint),
    (2, 1::smallint)
) AS v(store_id, position)
  ON (u.phone = '13400000000' OR u.email = '13400000000@dev.test.local')
 AND u.deleted_at IS NULL
ON CONFLICT (user_id, store_id) DO UPDATE SET
  position = EXCLUDED.position,
  updated_at = now();

UPDATE users
SET current_store_id = 1
WHERE (phone = '13400000000' OR email = '13400000000@dev.test.local')
  AND deleted_at IS NULL
  AND (current_store_id IS NULL OR current_store_id = 0);
