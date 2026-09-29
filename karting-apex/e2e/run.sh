#!/usr/bin/env bash
# E2E «Апекса»: Postgres + API в docker compose, имитатор Telegram Bot API на хосте,
# веб-приложение в headless Chrome. Один и тот же скрипт в CI и локально.
#
#   karting-apex/e2e/run.sh
#
# Переменные:
#   SKIP_WEB_BUILD=1  не пересобирать веб, если сборка уже есть (локально экономит ~8 минут)
#   KEEP=1            не останавливать стенд после прогона (для отладки)
#   CHROME=<путь>     свой Chrome/Chromium, если не нашёлся сам
#
# Результат: код выхода 0 — всё прошло. Логи API, имитатора и скриншоты — в e2e/out/.
set -uo pipefail

E2E="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APEX="$(dirname "$E2E")"
OUT="$E2E/out"
DIST="$APEX/client/webApp/build/dist/wasmJs/productionExecutable"
PROJECT=apexe2e
export E2E_DB_CONTAINER="$PROJECT-db-1" E2E_OUT="$OUT" PYTHONIOENCODING=utf-8 CDP_PORT="${CDP_PORT:-9333}"

# Windows (Git Bash): нативным программам нужны пути вида C:/..., а не /c/...
native() { if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s\n' "$1"; fi; }
compose() { (cd "$APEX/backend" && docker compose -p "$PROJECT" -f compose.yaml -f ../e2e/compose.e2e.yaml --profile app "$@"); }
section() { printf '\n=== %s ===\n' "$1"; }

PY=""
for candidate in python3 python; do
  # На Windows python3 бывает заглушкой Microsoft Store — проверяем, что он запускается.
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c 'import sys' >/dev/null 2>&1; then PY="$candidate"; break; fi
done
[ -n "$PY" ] || { echo "Нужен Python 3"; exit 2; }

if [ -z "${CHROME:-}" ]; then
  for candidate in google-chrome google-chrome-stable chromium chromium-browser \
                   "/c/Program Files/Google/Chrome/Application/chrome.exe"; do
    if command -v "$candidate" >/dev/null 2>&1 || [ -x "$candidate" ]; then CHROME="$candidate"; break; fi
  done
fi
[ -n "${CHROME:-}" ] || { echo "Не найден Chrome: задайте CHROME=<путь>"; exit 2; }

mkdir -p "$OUT"
rm -f "$OUT"/*.png "$OUT"/*.log
PROFILE="$(mktemp -d)"
pids=()

cleanup() {
  local status=$?
  compose logs --no-color api > "$OUT/api.log" 2>&1 || true
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
  if [ "${KEEP:-0}" != 1 ]; then compose down -v --remove-orphans > /dev/null 2>&1 || true; fi
  rm -rf "$PROFILE" 2>/dev/null || true
  exit "$status"
}
trap cleanup EXIT

wait_http() { # url, секунд
  local i
  for i in $(seq 1 "$2"); do curl -sf -o /dev/null "$1" && return 0; sleep 1; done
  echo "Не дождались $1"; return 1
}

section "Стенд: имитатор Telegram, Postgres, API"
"$PY" "$(native "$E2E/fake_telegram.py")" > "$OUT/fake_telegram.log" 2>&1 &
pids+=($!)
wait_http http://127.0.0.1:8099/_state 15 || exit 1
compose up -d --build || exit 1
wait_http http://127.0.0.1:8090/healthz 120 || exit 1

failed=0
section "Бот (вебхук, команды, отмена кнопкой, лист ожидания)"
(cd "$E2E" && "$PY" bot_e2e.py) || failed=1

section "Веб: сборка"
if [ "${SKIP_WEB_BUILD:-0}" = 1 ] && [ -f "$DIST/index.html" ]; then
  echo "SKIP_WEB_BUILD=1 — беру готовую сборку"
else
  (cd "$APEX/client" && ./gradlew :webApp:wasmJsBrowserDistribution --console=plain) || exit 1
fi

section "Веб: проверки в headless Chrome"
"$PY" -m http.server 8795 --bind 127.0.0.1 --directory "$(native "$DIST")" > "$OUT/web.log" 2>&1 &
pids+=($!)
chrome_flags=(--headless=new "--remote-debugging-port=$CDP_PORT" "--user-data-dir=$(native "$PROFILE")"
              --no-first-run --no-default-browser-check)
# На раннере CI песочница Chrome может не запуститься (Ubuntu 24.04 ограничивает user namespaces);
# раннер одноразовый, поэтому там без неё. На машине разработчика песочница остаётся.
[ "${CI:-}" = "true" ] && chrome_flags+=(--no-sandbox)
"$CHROME" "${chrome_flags[@]}" about:blank > "$OUT/chrome.log" 2>&1 &
pids+=($!)
wait_http http://127.0.0.1:8795/ 15 || exit 1
wait_http "http://127.0.0.1:$CDP_PORT/json/version" 30 || exit 1
(cd "$E2E" && "$PY" ui_e2e.py) || failed=1

section "Итог"
if [ "$failed" = 0 ]; then echo "E2E: всё прошло"; else echo "E2E: есть упавшие проверки (логи и скриншоты — в e2e/out/)"; fi
exit "$failed"
