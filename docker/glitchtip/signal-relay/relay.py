"""
Translates a Glitchtip generic-webhook alert payload into a Signal message and
sends it through signal-cli-rest-api. Glitchtip's payload shape (see
apps/alerts/webhooks.py in glitchtip-backend) is:

    {
      "text": "Glitchtip Alert",
      "attachments": [
        {
          "title": "<issue title>",
          "title_link": "<url to the issue>",
          "text": "<culprit>",
          "color": "#rrggbb",
          "fields": [{"title": "Project", "value": "augustin", "short": true}, ...]
        }
      ]
    }

No third-party dependencies so the image can stay a plain python:alpine with
nothing to pip install.
"""

import json
import os
import sys
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SIGNAL_API_URL = os.environ.get("SIGNAL_API_URL", "http://signal-cli-rest-api:8080")
SIGNAL_SENDER_NUMBER = os.environ.get("SIGNAL_SENDER_NUMBER", "").strip()
SIGNAL_RECIPIENTS = [
    r.strip() for r in os.environ.get("SIGNAL_RECIPIENTS", "").split(",") if r.strip()
]
PORT = int(os.environ.get("PORT", "8080"))


def format_message(payload: dict) -> str:
    lines = [payload.get("text") or "Glitchtip Alert"]
    for attachment in payload.get("attachments") or []:
        lines.append("")
        if attachment.get("title"):
            lines.append(attachment["title"])
        if attachment.get("text"):
            lines.append(attachment["text"])
        fields = attachment.get("fields") or []
        if fields:
            lines.append(
                ", ".join(f"{f.get('title')}: {f.get('value')}" for f in fields)
            )
        if attachment.get("title_link"):
            lines.append(attachment["title_link"])
    return "\n".join(lines)


def send_to_signal(message: str) -> None:
    if not SIGNAL_SENDER_NUMBER or not SIGNAL_RECIPIENTS:
        print(
            "glitchtip-signal-relay: SIGNAL_SENDER_NUMBER/SIGNAL_RECIPIENTS not "
            "configured, dropping alert",
            file=sys.stderr,
        )
        return

    body = json.dumps(
        {
            "message": message,
            "number": SIGNAL_SENDER_NUMBER,
            "recipients": SIGNAL_RECIPIENTS,
        }
    ).encode()
    req = urllib.request.Request(
        f"{SIGNAL_API_URL}/v2/send",
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            print(f"glitchtip-signal-relay: sent alert, signal-cli responded {resp.status}")
    except urllib.error.HTTPError as e:
        print(
            f"glitchtip-signal-relay: signal-cli rejected the message: "
            f"{e.code} {e.read().decode(errors='replace')}",
            file=sys.stderr,
        )
    except urllib.error.URLError as e:
        print(f"glitchtip-signal-relay: could not reach signal-cli-rest-api: {e}", file=sys.stderr)


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        print(f"glitchtip-signal-relay: {self.address_string()} {fmt % args}")

    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path != "/webhook":
            self.send_response(404)
            self.end_headers()
            return

        length = int(self.headers.get("Content-Length", 0))
        raw = self.rfile.read(length) if length else b""
        try:
            payload = json.loads(raw or b"{}")
        except json.JSONDecodeError:
            print(f"glitchtip-signal-relay: ignoring non-JSON body: {raw!r}", file=sys.stderr)
            payload = {}

        send_to_signal(format_message(payload))

        # Always 200 — a failed relay/signal-cli hop must not make Glitchtip
        # treat this as a delivery error and retry-storm; failures are logged above.
        self.send_response(200)
        self.end_headers()


if __name__ == "__main__":
    print(f"glitchtip-signal-relay: listening on :{PORT}, forwarding to {SIGNAL_API_URL}")
    ThreadingHTTPServer(("0.0.0.0", PORT), Handler).serve_forever()
