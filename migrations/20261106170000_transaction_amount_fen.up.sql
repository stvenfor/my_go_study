-- harden-h6b-transaction-money-unit：transactions 金额存储收敛到整数分。
--
-- 取证（docs/money-unit-audit.md §3 / §6.2）：
--   transactions.amount 是**裸 numeric（无精度无标度）**（见 20260923167500_align_gorm_only_schema.up.sql:40），
--   PG 会照收客户端 float 累加残值 0.30000000000000004，而全链路（DTO → handler → repo）零 Round。
--   所以本迁移的目标不是「float 换成分」，而是「存储侧不再能容纳 >2 位小数的脏值，且口径唯一」。
--
-- 本文件只做五步里的**步 1 加列 + 步 2 回填**：旧列 amount 不删（步 5 属后续轮，
-- 且删列前必须先确认双写无漂移 —— 留旧列也是本迁移可安全 down 的前提）。
--
-- 实测输出（三态：干净 / 亚分脏值 / 负数）见
-- docs/acceptance-records/2026-11-06-harden-h6b-transaction-money-unit.md，不在本注释里复述数字。

-- 步 1：加列。先可空，回填完成后才收紧为 NOT NULL。
--   可空 + Go 侧 *int64 的组合，保证「库尚未迁移/未回填」时读路径回落旧列，不会把 NULL 读成 0 元。
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS amount_fen bigint;

-- 硬门槛（Brief 验收第 3 条）：**换算是精确的还是吃掉钱，必须在回填前判定，不能事后发现。**
--   主判据用 1e-9 容差（分尺度）而不是 `<>`，两类的区别是量级的：
--     amount=0.30000000000000004 → amount*100=30.00000000000000400，与 30 差 4e-15 → **放过**
--       （这只是 IEEE754 表示残值，round 回 30 分，一分钱没丢）；
--     amount=1.005               → amount*100=100.500，与 101 差 0.5 → **拦住**
--       （真·亚分金额，round 会凭空多/少 0.5 分）。
--   h6 记录里那条更严的 `amount <> round(amount,2)` 会把两类都拦下，这里只作为附带计数报出来便于人工核对，
--   不作为放行条件（否则合法数据被 float 残值卡死，人无法处理）。
--   非 0 行 → RAISE EXCEPTION。golang-migrate 把整个 up 文件包在一个事务里，中止即全部回滚，
--   连 amount_fen 列本身都不残留（这条在 acceptance-record 用 \d 输出验证）。
--   代价：中止会把 schema_migrations 留在 dirty=true，**处理完脏数据后必须先复位版本号才能重跑**：
--     make migrate-force VERSION=20261104160000 && make migrate-up
--   （直接 make migrate-up 会得到 "Dirty database version 20261106170000. Fix and force version."，
--    这是实测输出，不是推测；详见 acceptance-record 组 D。）
DO $$
DECLARE
  n_blocking bigint;
  n_strict   bigint;
  n_negative bigint;
BEGIN
  SELECT count(*) INTO n_blocking
  FROM   transactions
  WHERE  amount IS NOT NULL
    AND  abs(amount * 100 - round(amount * 100)) > 1e-9;

  SELECT count(*) INTO n_strict
  FROM   transactions
  WHERE  amount IS NOT NULL
    AND  amount <> round(amount, 2);

  IF n_blocking > 0 THEN
    RAISE EXCEPTION
      '回填门槛未过：% 行金额换成分会改变数值（真·亚分脏值）。附带计数：更严口径 amount <> round(amount, 2) 下有 % 行（含这 % 行），'
      '其余属 IEEE754 表示残值，round 回分不丢钱。'
      '禁止用 round/trunc 吃掉钱 —— 先人工核对样本：'
      'SELECT id, user_id, amount FROM transactions WHERE amount IS NOT NULL AND abs(amount*100 - round(amount*100)) > 1e-9 ORDER BY id DESC LIMIT 50;'
      '处理完再重跑本迁移（中止已把 schema_migrations 置 dirty，先 make migrate-force VERSION=20261104160000 复位）。', n_blocking, n_strict, n_blocking;
  END IF;

  -- 负数口径检查（Brief 验收第 2 条）：本表的正负由 type（income/expense）表达，amount 存绝对值。
  -- 若库里确有负数，说明口径与本 CHECK 冲突，同样要人工裁决，不许静默把约束降级。
  SELECT count(*) INTO n_negative FROM transactions WHERE amount < 0;
  IF n_negative > 0 THEN
    RAISE EXCEPTION
      'transactions 有 % 行负数 amount，与 CHECK (amount_fen >= 0) 冲突（负数应由 type 表达）。'
      '请人工确认口径：改数，或把本迁移的约束降级并在 acceptance-record 说明理由。', n_negative;
  END IF;
END $$;

-- 步 2：回填。门槛已过 → 此处的 round 只用来抹平 ≤1e-9 的 IEEE754 表示残值（30.00000000000000400 → 30），
-- 会改变真实金额的行在上面已被拦住，不存在「悄悄吃掉钱」。
UPDATE transactions
SET    amount_fen = round(amount * 100)::bigint
WHERE  amount_fen IS NULL;

ALTER TABLE transactions ALTER COLUMN amount_fen SET NOT NULL;

-- 非负约束：先 DROP IF EXISTS 再 ADD，让本迁移在「AutoMigrate 先建过列」的库上也能重放。
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_amount_fen_nonneg;
ALTER TABLE transactions ADD CONSTRAINT transactions_amount_fen_nonneg CHECK (amount_fen >= 0);

COMMENT ON COLUMN transactions.amount_fen IS
  '金额整数分（harden-h6b 的存储口径）；对外 JSON 仍是 amount 元，由本列换算而来。旧列 amount 保留做双写与漂移核对。';
