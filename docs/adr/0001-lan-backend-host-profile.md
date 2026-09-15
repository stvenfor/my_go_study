# Use a dedicated LAN profile for home Wi‑Fi true-device联调

We need the Mac to act as a LAN Backend Host for iOS/Android devices without treating that posture as either pure local `dev` (test OTP and debug defaults on the home LAN) or `prod` (too strict for on-demand Compose联调). Decision: introduce `APP_ENV=lan` / `config.lan.yaml`, keep real `PINNED_LAN_IP` and lab whitelist only in local env, and default Compose LAN overlay to publish only port 8080.
