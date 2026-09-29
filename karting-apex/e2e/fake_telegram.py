"""Имитатор Telegram Bot API для E2E: запоминает всё, что «отправил» бот.

GET /_state — сообщения, вебхук, меню команд, ответы на нажатия кнопок.
API ходит сюда вместо api.telegram.org по TELEGRAM_API_BASE (см. compose.e2e.yaml).
"""
import json
import os
import re
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

STATE = {"webhook": None, "messages": [], "calls": [], "commands": [], "answerCallbackQuery": [], "editMessageReplyMarkup": []}
TOKEN = "local-test-token"  # TELEGRAM_BOT_TOKEN в compose.e2e.yaml
PORT = int(os.environ.get("FAKE_TELEGRAM_PORT", "8099"))


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def _send(self, code, payload):
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        self._send(200, STATE) if self.path == "/_state" else self._send(404, {"ok": False})

    def do_POST(self):
        match = re.fullmatch(r"/bot([^/]+)/(\w+)", self.path)
        length = int(self.headers.get("Content-Length") or 0)
        payload = json.loads(self.rfile.read(length) or b"{}")
        if not match:
            self._send(404, {"ok": False, "description": "Not Found"})
            return
        token, method = match.groups()
        STATE["calls"].append(method)
        if token != TOKEN:
            self._send(401, {"ok": False, "error_code": 401, "description": "Unauthorized"})
        elif method == "getMe":
            self._send(200, {"ok": True, "result": {"id": 1, "is_bot": True, "username": "apex_local_test_bot"}})
        elif method == "setWebhook":
            STATE["webhook"] = payload
            self._send(200, {"ok": True, "result": True})
        elif method == "setMyCommands":
            STATE["commands"].append(payload)
            self._send(200, {"ok": True, "result": True})
        elif method in ("answerCallbackQuery", "editMessageReplyMarkup"):
            STATE[method].append(payload)
            self._send(200, {"ok": True, "result": True})
        elif method == "sendMessage":
            STATE["messages"].append(payload)
            self._send(200, {"ok": True, "result": {"message_id": len(STATE["messages"])}})
        else:
            self._send(400, {"ok": False, "description": "unsupported"})


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", PORT), Handler).serve_forever()
