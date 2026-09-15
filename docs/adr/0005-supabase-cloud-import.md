# Import from Supabase Cloud into local Postgres

Cloud GoTrue password hashes are not available via the Admin API, so a one-shot import cannot preserve login passwords. Decision: ship `cmd/import-supabase` that copies Auth user UUIDs/emails, profiles, and transactions via service_role, and sets all imported local users to a caller-supplied `--default-password` (bcrypt). Upsert by id for idempotent re-runs.
