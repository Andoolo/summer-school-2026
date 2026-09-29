"""Общее для E2E «Апекса»: запросы к API, SQL в контейнере базы, проверки и итог."""
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request

API = os.environ.get("E2E_API", "http://127.0.0.1:8090")
FAKE_TELEGRAM = os.environ.get("E2E_FAKE_TELEGRAM", "http://127.0.0.1:8099")
DB_CONTAINER = os.environ.get("E2E_DB_CONTAINER", "apexe2e-db-1")

_failures = 0


def check(name, ok, detail=""):
    """Печатает OK/FAIL и продолжает: так за один прогон видны все упавшие проверки."""
    global _failures
    print(("OK   " if ok else "FAIL ") + name + ("" if ok else f"  -> {detail}"), flush=True)
    _failures += 0 if ok else 1


def finish():
    print("ИТОГ:", "ВСЁ ПРОШЛО" if _failures == 0 else f"упало {_failures}", flush=True)
    sys.exit(1 if _failures else 0)


def call(method, path, body=None, token=None, base=API, headers=None):
    """HTTP-запрос к API; возвращает (статус, JSON или None) и для ошибок тоже."""
    req = urllib.request.Request(base + path, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    for key, value in (headers or {}).items():
        req.add_header(key, value)
    data = json.dumps(body).encode() if body is not None else None
    try:
        with urllib.request.urlopen(req, data, timeout=20) as resp:
            raw = resp.read()
            return resp.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as err:
        raw = err.read()
        return err.code, (json.loads(raw) if raw else None)


def psql(sql):
    out = subprocess.run(["docker", "exec", "-i", DB_CONTAINER, "psql", "-U", "volna", "-d", "volna", "-At", "-c", sql],
                         capture_output=True, text=True, encoding="utf-8")
    if out.returncode != 0:
        raise RuntimeError(out.stderr)
    return out.stdout.strip()


def wait_for(predicate, seconds=10.0, step=0.3):
    deadline = time.time() + seconds
    while time.time() < deadline:
        if predicate():
            return True
        time.sleep(step)
    return False


def login_by_code(phone):
    """Вход по SMS-коду: в режиме разработки API возвращает код прямо в ответе."""
    status, body = call("POST", "/auth/request-code", {"phone": phone})
    assert status == 200, (status, body)
    status, body = call("POST", "/auth/verify-code", {"phone": phone, "code": body["code"]})
    assert status == 200, (status, body)
    return body["token"]


def link_telegram(phone, chat_id):
    """Привязывает чат к клиенту напрямую в базе: в очередь можно только с Telegram."""
    psql(f"UPDATE clients SET telegram_chat_id = NULL WHERE telegram_chat_id = {chat_id}")
    psql(f"UPDATE clients SET telegram_chat_id = {chat_id}, telegram_notifications = true WHERE phone = '{phone}'")


def free_queue(*slot_ids):
    """Закрывает чужие очереди на заездах сценария: предложение места получает первый в
    очереди, и оставшаяся от другой проверки запись увела бы его у проверяемого."""
    ids = "','".join(slot_ids)
    psql(f"UPDATE waitlist_entries SET status = 'left', closed_at = now() "
         f"WHERE slot_id IN ('{ids}') AND status IN ('waiting', 'notified')")


def unique_suffix():
    return str(int(time.time()))[-5:]
