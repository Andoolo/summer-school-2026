"""Веб-приложение в headless Chrome (через cdp.mjs): то, что видит человек.

- каталог: на заезде, где человек в очереди, — «Мест нет · 2-й в очереди», на другом
  заполненном — «Мест нет · есть лист ожидания»;
- экран заезда: срок предложения приходит с сервера («На запись будет 15 минут.»);
- сессия истекла на экране заезда → вход → тот же заезд.

Нужны: API на :8090 с ботом, веб на :8795, Chrome с отладкой на CDP_PORT (по умолчанию 9333).
"""
import json
import os
import pathlib
import subprocess
import time

from common import call, check, finish, free_queue, link_telegram, login_by_code, psql, unique_suffix

HERE = pathlib.Path(__file__).parent
OUT = pathlib.Path(os.environ.get("E2E_OUT", HERE / "out"))
WEB = os.environ.get("E2E_WEB", "http://127.0.0.1:8795")


def cdp(*args):
    out = subprocess.run(["node", str(HERE / "cdp.mjs"), *args], capture_output=True, text=True, encoding="utf-8")
    if out.returncode != 0:
        raise RuntimeError(out.stderr)
    return out.stdout.strip()


def page_text():
    # Compose для веба строит дерево доступности в shadowRoot: тексты — в textContent и aria-label.
    raw = cdp("eval", "(document.body.shadowRoot.textContent || '') + ' | ' + "
                      "[...document.body.shadowRoot.querySelectorAll('*')].map(e => e.getAttribute('aria-label') || '')"
                      ".filter(Boolean).join(' | ')")
    return json.loads(raw)


def open_as(token, url_suffix, tag):
    """Открывает приложение под сессией token: токен кладётся туда же, куда его пишет вход."""
    cdp("goto", f"{WEB}/?r={tag}")
    cdp("eval", f"localStorage.setItem('volna_bearer_token', '{token}'); 'ok'")
    cdp("goto", f"{WEB}/?r={tag}b{url_suffix}")


def shot(name):
    OUT.mkdir(parents=True, exist_ok=True)
    cdp("shot", str(OUT / name))


suffix = unique_suffix()

# --- каталог: своя очередь и общая подпись ---
first_phone, me_phone = f"+7999{suffix}41", f"+7999{suffix}42"
first, me = login_by_code(first_phone), login_by_code(me_phone)
link_telegram(first_phone, int("81" + suffix[-4:]))
link_telegram(me_phone, int("82" + suffix[-4:]))
catalog_slots = psql("SELECT id FROM slots WHERE start_at > now() + interval '1 hour' AND status = 'scheduled' "
                     "ORDER BY start_at, id LIMIT 2").split()
free_queue(*catalog_slots)
psql("UPDATE slots SET free_seats = 0 WHERE id IN ('" + "','".join(catalog_slots) + "')")
for token in (first, me):
    status, entry = call("POST", f"/slots/{catalog_slots[0]}/waitlist", {"seats_count": 1}, token)
    assert status == 201, (status, entry)

open_as(me, "", suffix + "cat")
seen = ""
for _ in range(10):
    seen += " || " + page_text()
    if "2-й в очереди" in seen and "есть лист ожидания" in seen:
        break
    cdp("scroll", "195", "600", "600")
shot("catalog.png")
check("каталог: своя очередь — «Мест нет · 2-й в очереди»", "Мест нет · 2-й в очереди" in seen, seen[-600:])
check("каталог: другой заполненный — «Мест нет · есть лист ожидания»", "Мест нет · есть лист ожидания" in seen, seen[-600:])

# --- экран заезда: срок предложения с сервера ---
phone = f"+7999{suffix}31"
token = login_by_code(phone)
link_telegram(phone, int("78" + suffix[-4:]))
excluded = "','".join(catalog_slots)
slot = psql("SELECT id FROM slots WHERE start_at > now() + interval '6 hours' AND status = 'scheduled' "
            f"AND id NOT IN ('{excluded}') ORDER BY start_at, id LIMIT 1")
free_queue(slot)
psql(f"UPDATE slots SET free_seats = 0 WHERE id = '{slot}'")
open_as(token, f"#slot/{slot}", suffix + "slot")
for _ in range(3):
    cdp("scroll", "195", "600", "1500")
shot("slot-waitlist.png")
text = page_text()
check("заезд: «На запись будет 15 минут.» (срок с сервера)", "На запись будет 15 минут." in text, text[-400:])

# --- сессия истекла на экране заезда → вход → тот же заезд ---
psql(f"UPDATE auth_sessions SET revoked_at = now() WHERE client_id = (SELECT id FROM clients WHERE phone = '{phone}')")
cdp("goto", f"{WEB}/?r={suffix}exp#slot/{slot}")
hash_after_401 = json.loads(cdp("eval", "location.hash"))
check("истёкшая сессия ведёт на вход", hash_after_401 == "#auth", hash_after_401)
clicked = cdp("eval", "(() => { const b = [...document.body.shadowRoot.querySelectorAll('[role=\"button\"], button')]"
                      ".find(e => (e.getAttribute('aria-label') || e.textContent || '').includes('без регистрации'));"
                      " if (!b) return 'no demo button'; b.click(); return 'clicked'; })()")
final_hash = ""
for _ in range(20):
    time.sleep(1)
    final_hash = json.loads(cdp("eval", "location.hash"))
    if final_hash == f"#slot/{slot}":
        break
shot("after-login.png")
check("после входа — тот же заезд", final_hash == f"#slot/{slot}", (clicked, final_hash))

finish()
