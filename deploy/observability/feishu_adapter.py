#!/usr/bin/env python3
"""Alertmanager → 飞书自定义机器人 webhook 适配（无 URL 时吞掉告警）。"""
from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer


FEISHU_URL = os.environ.get("FEISHU_ALERT_WEBHOOK_URL", "").strip()
PORT = int(os.environ.get("FEISHU_ADAPTER_PORT", "8089"))


def format_text(payload: dict) -> str:
    status = payload.get("status", "unknown")
    alerts = payload.get("alerts") or []
    lines = [f"[Alertmanager] status={status} count={len(alerts)}"]
    for a in alerts[:10]:
        labels = a.get("labels") or {}
        ann = a.get("annotations") or {}
        name = labels.get("alertname", "?")
        sev = labels.get("severity", "")
        summary = ann.get("summary") or ann.get("description") or ""
        lines.append(f"- {a.get('status', status)} {name}({sev}): {summary}")
    if len(alerts) > 10:
        lines.append(f"... and {len(alerts) - 10} more")
    return "\n".join(lines)


def post_feishu(text: str) -> None:
    body = json.dumps({"msg_type": "text", "content": {"text": text}}).encode()
    req = urllib.request.Request(
        FEISHU_URL,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        resp.read()


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args) -> None:  # quieter
        return

    def do_GET(self) -> None:
        if self.path in ("/health", "/"):
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
            return
        self.send_response(404)
        self.end_headers()

    def do_POST(self) -> None:
        if self.path != "/webhook":
            self.send_response(404)
            self.end_headers()
            return
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length else b"{}"
        if not FEISHU_URL:
            self.send_response(204)
            self.end_headers()
            return
        try:
            payload = json.loads(raw.decode() or "{}")
            post_feishu(format_text(payload))
            self.send_response(204)
            self.end_headers()
        except (json.JSONDecodeError, urllib.error.URLError, TimeoutError, OSError) as e:
            self.send_response(502)
            self.end_headers()
            self.wfile.write(str(e).encode())


def main() -> None:
    server = HTTPServer(("0.0.0.0", PORT), Handler)
    print(f"feishu-adapter listening on :{PORT} webhook_configured={bool(FEISHU_URL)}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
