# Local Auth + Postgres with dual provider switch

Day-to-day运行应脱离 Supabase Cloud，但仍要能迁回 Cloud。Decision: implement `auth.provider=local|supabase` — local uses UUID `auth_users` + app-owned JWT/refresh and GORM/SQL for profiles/transactions; supabase keeps existing GoTrue + PostgREST. Upgrade `APP_ENV=lan` to mean local full stack on the LAN (not Cloud-over-LAN). Skip local Postgres RLS for now; keep UUID schema aligned with Cloud.
