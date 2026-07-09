-- 一键启用 transactions 表 RLS（在 Supabase Dashboard → SQL Editor 执行）
-- 问题：未执行此脚本时，任意登录用户可读写他人 transactions

-- 1. 确保 user_id 列存在
alter table public.transactions
  add column if not exists user_id uuid references auth.users(id) on delete cascade;

create index if not exists transactions_user_id_idx on public.transactions(user_id);

-- 2. 启用 RLS
alter table public.transactions enable row level security;

-- 3. 删除旧策略（若存在）并重建（仅 authenticated 角色）
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

-- 4. 验证（应返回 rls_enabled = true，policy_count = 4）
select
  c.relname as table_name,
  c.relrowsecurity as rls_enabled,
  (select count(*) from pg_policies p where p.tablename = c.relname) as policy_count
from pg_class c
join pg_namespace n on n.oid = c.relnamespace
where n.nspname = 'public' and c.relname = 'transactions';
