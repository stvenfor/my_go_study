# Prefer cleartext HTTP/WS on trusted LAN; defer TLS

Phones on the same Wi‑Fi must reach the LAN Backend Host with minimal friction (both iOS and Android). Full HTTPS/wss now forces certificates and a reverse proxy before联调 works. Decision: ship cleartext `http`/`ws` aligned on a pinned LAN IP first; document Caddy/Nginx TLS termination as a later appendix without blocking the current path.
