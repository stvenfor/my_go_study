-- 清理 transactions 表全部 RLS 策略并重建（解决 policy_count=8 仍泄露数据的问题）
-- 原因：存在旧的宽松策略，PostgreSQL 多条 SELECT 策略是 OR 关系，有一条放行即可见全部数据
-- 在 Supabase Dashboard → SQL Editor 执行

-- 1. 查看当前所有策略（执行后可检查）
select policyname, cmd, roles, qual, with_check
from pg_policies
where schemaname = 'public' and tablename = 'transactions'
order by policyname;

-- 2. 删除 transactions 表上的全部策略
do $$
declare pol record;
begin
  for pol in
    select policyname
    from pg_policies
    where schemaname = 'public' and tablename = 'transactions'
  loop
    execute format('drop policy if exists %I on public.transactions', pol.policyname);
  end loop;
end $$;

-- 3. 确保 RLS 开启
alter table public.transactions enable row level security;

-- 4. 仅重建 4 条策略（authenticated 用户只能操作自己的行）
create policy "transactions_select_own"
  on public.transactions for select
  to authenticated
  using (auth.uid() = user_id);

create policy "transactions_insert_own"
  on public.transactions for insert
  to authenticated
  with check (auth.uid() = user_id);

create policy "transactions_update_own"
  on public.transactions for update
  to authenticated
  using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

create policy "transactions_delete_own"
  on public.transactions for delete
  to authenticated
  using (auth.uid() = user_id);

-- 5. 验证（应 rls_enabled=true, policy_count=4）
select
  c.relname as table_name,
  c.relrowsecurity as rls_enabled,
  (select count(*) from pg_policies p where p.tablename = c.relname) as policy_count
from pg_class c
join pg_namespace n on n.oid = c.relnamespace
where n.nspname = 'public' and c.relname = 'transactions';

-- 6. 再次查看策略列表（应只有 4 条）
select policyname, cmd, roles
from pg_policies
where schemaname = 'public' and tablename = 'transactions'
order by policyname;
