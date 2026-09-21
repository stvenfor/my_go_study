-- 门店组织与权限目录。可重复执行。启动路径与 migrations 使用同一份。
-- 平台分配 store_id 为空，店内分配必须有店。对照 role.scope 由写入方保证，CHECK 不能跨表。

CREATE TABLE IF NOT EXISTS wys_store (
  store_id   integer PRIMARY KEY,
  name       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_wys_store_id CHECK (store_id > 0)
);

COMMENT ON TABLE wys_store IS '门店。多名用户可同属一家，不是个人统计卡';
COMMENT ON COLUMN wys_store.store_id IS '店铺 ID，正整数';

CREATE TABLE IF NOT EXISTS role (
  code  varchar(64) PRIMARY KEY,
  scope text NOT NULL,
  name  varchar(128) NOT NULL,
  CONSTRAINT chk_role_scope CHECK (scope IN ('platform', 'store'))
);

COMMENT ON TABLE role IS '权限角色定义。运行时不新建、不改绑定';
COMMENT ON COLUMN role.scope IS 'platform=不挂店 store=必须挂店';

CREATE TABLE IF NOT EXISTS permission (
  code varchar(64) PRIMARY KEY,
  name varchar(128) NOT NULL
);

COMMENT ON TABLE permission IS '能力码。范围写在码的含义里，只有授予没有拒绝';

CREATE TABLE IF NOT EXISTS role_permission (
  role_code       varchar(64) NOT NULL REFERENCES role (code) ON DELETE RESTRICT,
  permission_code varchar(64) NOT NULL REFERENCES permission (code) ON DELETE RESTRICT,
  PRIMARY KEY (role_code, permission_code)
);

CREATE TABLE IF NOT EXISTS wys_store_member (
  user_id    varchar(64) NOT NULL REFERENCES users (user_id) ON DELETE RESTRICT,
  store_id   integer NOT NULL REFERENCES wys_store (store_id) ON DELETE RESTRICT,
  position   smallint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, store_id),
  CONSTRAINT chk_wys_store_member_position CHECK (position IN (0, 1, 2))
);

COMMENT ON TABLE wys_store_member IS '门店成员。一人一店一个职务，职务不是权限';
COMMENT ON COLUMN wys_store_member.position IS '0=销售顾问 1=销售经理 2=总经理';

CREATE TABLE IF NOT EXISTS user_role (
  user_id    varchar(64) NOT NULL REFERENCES users (user_id) ON DELETE RESTRICT,
  role_code  varchar(64) NOT NULL REFERENCES role (code) ON DELETE RESTRICT,
  store_id   integer NULL REFERENCES wys_store (store_id) ON DELETE RESTRICT,
  granted_by varchar(64) NULL REFERENCES users (user_id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE user_role IS '角色分配。平台行 store_id 为空；店内行必须有店';
COMMENT ON COLUMN user_role.granted_by IS '授予人。种子为空；授予人被删时置空';

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_role_platform
  ON user_role (user_id, role_code) WHERE store_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_role_store
  ON user_role (user_id, role_code, store_id) WHERE store_id IS NOT NULL;

INSERT INTO role (code, scope, name) VALUES
  ('platform_admin', 'platform', '平台管理员'),
  ('store_admin', 'store', '店内管理员'),
  ('store_staff', 'store', '店员')
ON CONFLICT (code) DO NOTHING;

INSERT INTO permission (code, name) VALUES
  ('store.create', '建店'),
  ('member.write', '管理成员'),
  ('role.assign_store', '分配店内角色'),
  ('role.assign_platform', '分配平台角色'),
  ('profile.read', '读资料'),
  ('transaction.read_own', '读自己的收支'),
  ('transaction.write_own', '写自己的收支'),
  ('transaction.read_store', '读本店收支')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permission (role_code, permission_code) VALUES
  ('platform_admin', 'store.create'),
  ('platform_admin', 'member.write'),
  ('platform_admin', 'role.assign_store'),
  ('platform_admin', 'role.assign_platform'),
  ('platform_admin', 'profile.read'),
  ('platform_admin', 'transaction.read_own'),
  ('platform_admin', 'transaction.write_own'),
  ('platform_admin', 'transaction.read_store'),
  ('store_admin', 'member.write'),
  ('store_admin', 'role.assign_store'),
  ('store_admin', 'profile.read'),
  ('store_admin', 'transaction.read_own'),
  ('store_admin', 'transaction.write_own'),
  ('store_admin', 'transaction.read_store'),
  ('store_staff', 'profile.read'),
  ('store_staff', 'transaction.read_own'),
  ('store_staff', 'transaction.write_own')
ON CONFLICT (role_code, permission_code) DO NOTHING;

-- 新库直接建成每人每店一行。旧库已有「store_id 主键」时，下面的 DO 块再改。
CREATE TABLE IF NOT EXISTS wys_user_store_stats (
  user_id         varchar(64) NOT NULL,
  store_id        integer NOT NULL,
  store_name      text NOT NULL DEFAULT '',
  role            smallint NOT NULL DEFAULT 0,
  days_joined     integer NOT NULL DEFAULT 0,
  employee_count  integer NOT NULL DEFAULT 0,
  store_days      integer NOT NULL DEFAULT 0,
  total_customers integer NOT NULL DEFAULT 0,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, store_id),
  CONSTRAINT chk_wys_user_store_stats_store_id CHECK (store_id > 0),
  CONSTRAINT chk_wys_user_store_stats_role CHECK (role IN (0, 1, 2)),
  CONSTRAINT chk_wys_user_store_stats_nonneg CHECK (
    days_joined >= 0
    AND employee_count >= 0
    AND store_days >= 0
    AND total_customers >= 0
  ),
  CONSTRAINT wys_user_store_stats_store_id_fkey
    FOREIGN KEY (store_id) REFERENCES wys_store (store_id) ON DELETE RESTRICT
);

-- 已有统计卡：一店补一行门店，不把卡上的人写成成员。
INSERT INTO wys_store (store_id, name, created_at, updated_at)
SELECT DISTINCT ON (store_id)
  store_id,
  COALESCE(NULLIF(btrim(store_name), ''), '门店 ' || store_id::text),
  created_at,
  updated_at
FROM wys_user_store_stats
ORDER BY store_id, created_at
ON CONFLICT (store_id) DO NOTHING;

DO $$
DECLARE
  pk_cols text;
BEGIN
  IF to_regclass('wys_user_store_stats') IS NULL THEN
    RETURN;
  END IF;

  SELECT string_agg(a.attname, ',' ORDER BY k.ord)
  INTO pk_cols
  FROM pg_index i
  JOIN pg_class c ON c.oid = i.indrelid
  JOIN pg_namespace n ON n.oid = c.relnamespace
  JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, ord) ON true
  JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = k.attnum
  WHERE n.nspname = CURRENT_SCHEMA()
    AND c.relname = 'wys_user_store_stats'
    AND i.indisprimary;

  IF pk_cols IS DISTINCT FROM 'user_id,store_id' THEN
    EXECUTE (
      SELECT format('ALTER TABLE wys_user_store_stats DROP CONSTRAINT %I', conname)
      FROM pg_constraint
      WHERE conrelid = 'wys_user_store_stats'::regclass
        AND contype = 'p'
    );
    ALTER TABLE wys_user_store_stats ADD PRIMARY KEY (user_id, store_id);
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('wys_user_store_stats') IS NOT NULL
     AND NOT EXISTS (
       SELECT 1 FROM pg_constraint
       WHERE conname = 'wys_user_store_stats_store_id_fkey'
     ) THEN
    ALTER TABLE wys_user_store_stats
      ADD CONSTRAINT wys_user_store_stats_store_id_fkey
      FOREIGN KEY (store_id) REFERENCES wys_store (store_id) ON DELETE RESTRICT;
  END IF;
END $$;

COMMENT ON TABLE wys_user_store_stats IS '某人在某店的展示数字。每人每店一行，不是成员名单';
COMMENT ON COLUMN wys_user_store_stats.role IS '遗留职务列，资料接口不读。职务在 wys_store_member.position';

UPDATE users
SET current_store_id = NULL
WHERE current_store_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM wys_store s WHERE s.store_id = users.current_store_id
  );

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'users_current_store_id_fkey'
  ) THEN
    ALTER TABLE users
      ADD CONSTRAINT users_current_store_id_fkey
      FOREIGN KEY (current_store_id) REFERENCES wys_store (store_id) ON DELETE RESTRICT;
  END IF;
END $$;

COMMENT ON COLUMN users.current_store_id IS '当前店，必须是 wys_store.store_id。切换时还须已是该店成员';
