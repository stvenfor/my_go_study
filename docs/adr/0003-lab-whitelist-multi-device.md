# Lab whitelist for multi-device LAN联调; keep single-device default

iOS and Android must often stay signed in together against the same Mac backend, but product default is single-device session replacement. Globally disabling single-device on the LAN profile would diverge too far from production behaviour. Decision: keep single-device for normal accounts; put only lab test emails/user IDs in `AUTH_SESSION_WHITELIST_*` via local (untracked) env.
