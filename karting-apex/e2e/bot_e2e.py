"""Бот через настоящий вебхук: API + имитатор Telegram Bot API (fake_telegram.py).

Вход через Telegram, /start без ссылки, код сверки текстом, /stop и /notify, /whoami,
/stats (админ и не админ), группы и боты, кнопка «Отменить бронь» в два шага, меню
администратора, предложение места из листа ожидания со ссылкой на заезд.
"""
import time
import uuid

from common import FAKE_TELEGRAM, call, check, finish, free_queue, link_telegram, login_by_code, psql, unique_suffix, wait_for

ADMIN = 7301  # ADMIN_TELEGRAM_CHAT_ID в compose.e2e.yaml


def state():
    return call("GET", "/_state", base=FAKE_TELEGRAM)[1]


def to(chat):
    return [m["text"] for m in state()["messages"] if m["chat_id"] == chat]


def webhook(update):
    secret = state()["webhook"]["secret_token"]
    update["update_id"] = int(time.time() * 1000)
    status, _ = call("POST", "/telegram/webhook", update, headers={"X-Telegram-Bot-Api-Secret-Token": secret})
    assert status == 200, status


def say(chat, text="", contact=None, chat_type="private", is_bot=False):
    msg = {"message_id": 1, "from": {"id": chat, "first_name": "Анна", "is_bot": is_bot},
           "chat": {"id": chat, "type": chat_type}, "text": text}
    if contact:
        msg["contact"] = contact
    before = len(to(chat))
    webhook({"message": msg})
    wait_for(lambda: len(to(chat)) > before, 3)
    replies = to(chat)
    return replies[-1] if len(replies) > before else None


def press(chat, data):
    webhook({"callback_query": {"id": str(uuid.uuid4()), "from": {"id": chat, "first_name": "Анна"}, "data": data,
                                "message": {"message_id": 101, "chat": {"id": chat, "type": "private"}}}})
    return state()["answerCallbackQuery"][-1]


def book(slot, token):
    return call("POST", "/bookings", {"slot_id": slot, "seats_count": 1, "rental_count": 0}, token,
                headers={"Idempotency-Key": str(uuid.uuid4())})


# Бот подключается в фоне после старта API: ждём, пока он поставит вебхук.
check("бот подключился и поставил вебхук", wait_for(lambda: (state() or {}).get("webhook") is not None, 60), state())

suffix = unique_suffix()
chat = int("77" + suffix[-3:])
psql(f"UPDATE clients SET telegram_chat_id = NULL WHERE telegram_chat_id IN ({chat}, {ADMIN})")

# --- вход через Telegram ---
status, started = call("POST", "/auth/telegram/start")
check("старт входа через Telegram", status == 201 and started.get("deep_link"), (status, started))
param = started["deep_link"].split("start=")[1]
reply = say(chat, "/start " + param)
check("бот показывает код сверки", reply and started["confirm_code"] in reply, reply)
reply = say(chat, started["confirm_code"])
check("код сверки текстом — подсказка, а не вход", reply and "вводить не нужно" in reply, reply)
reply = say(chat, contact={"phone_number": f"7999{suffix}12", "first_name": "Анна", "user_id": chat})
check("контакт принят", reply and "Готово" in reply, reply)
status, poll = call("POST", "/auth/telegram/poll", {"poll_token": started["poll_token"]})
check("приложение получило сессию", status == 200 and poll.get("status") == "confirmed" and poll.get("token"), (status, poll))
token = poll.get("token")

# --- команды ---
reply = say(chat, "/start")
check("/start без ссылки — справка", reply and "бот приложения «Апекс»" in reply, reply)
reply = say(chat, "привет")
check("произвольный текст — справка", reply and "бот приложения «Апекс»" in reply, reply)
reply = say(chat, "/stop")
check("/stop отключает уведомления",
      reply and "отключены" in reply and psql(f"SELECT telegram_notifications FROM clients WHERE telegram_chat_id = {chat}") == "f", reply)
reply = say(chat, "/notify@apex_local_test_bot")
check("/notify@бот включает обратно",
      reply and "включены" in reply and psql(f"SELECT telegram_notifications FROM clients WHERE telegram_chat_id = {chat}") == "t", reply)
reply = say(chat, "/whoami")
check("/whoami", reply == f"Ваш Telegram chat id: {chat}", reply)
reply = say(chat, "/stats")
check("/stats не админу — справка", reply and "бот приложения «Апекс»" in reply, reply)
reply = say(ADMIN, "/stats")
check("/stats админу — сводка", reply and "Апекс — сводка" in reply, reply)
before = len(to(chat))
say(chat, "/stop", chat_type="group")
say(chat, "/stop", is_bot=True)
check("в группах и от ботов — тишина", len(to(chat)) == before, to(chat)[before:])

# --- кнопка «Отменить бронь» ---
slot = psql("SELECT id FROM slots WHERE start_at > now() + interval '6 hours' AND status = 'scheduled' ORDER BY start_at, id LIMIT 1")
free_queue(slot)
psql(f"UPDATE slots SET free_seats = greatest(free_seats, 3) WHERE id = '{slot}'")
status, booking = book(slot, token)
check("бронь создана", status == 201, (status, booking))
bid = booking["id"]
check("подтверждение с кнопкой пришло", wait_for(lambda: any("Бронь подтверждена" in t for t in to(chat))))
answer = press(chat, "cancel:" + bid)
check("«Отменить» спрашивает", answer.get("text") == "Точно отменить бронь?", answer)
answer = press(chat, "cancel_yes:" + bid)
check("«Да» отменяет", answer.get("text") == "Бронь отменена", answer)
_, got = call("GET", f"/bookings/{bid}", token=token)
check("в приложении бронь отменена", got["status"] == "cancelled", got)

# --- меню администратора и лист ожидания со ссылкой ---
menus = state()["commands"]
check("меню администратора с /stats",
      any(m.get("scope", {}).get("chat_id") == ADMIN and any(c["command"] == "stats" for c in m["commands"]) for m in menus), menus)

waiter_chat = chat + 1
waiter_phone = f"+7999{suffix}13"
waiter = login_by_code(waiter_phone)
link_telegram(waiter_phone, waiter_chat)
psql(f"UPDATE slots SET free_seats = 1 WHERE id = '{slot}'")
status, booking = book(slot, token)
check("последнее место занято", status == 201, (status, booking))
status, entry = call("POST", f"/slots/{slot}/waitlist", {"seats_count": 1}, waiter)
check("второй встал в очередь", status == 201, (status, entry))
status, _ = call("POST", f"/bookings/{booking['id']}/cancel", None, token)
check("бронь отменена в приложении", status == 200, status)
check("предложение со ссылкой на заезд",
      wait_for(lambda: any("Освободилось место" in t and f"#slot/{slot}" in t for t in to(waiter_chat))), to(waiter_chat))

finish()
