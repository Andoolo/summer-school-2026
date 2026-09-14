# Скриншоты картинг-приложения «Апекс»

Экраны по этапам — используются в [`../README.md`](../README.md) и в корневом README.

| Файлы | Этап |
|---|---|
| `login.png`, `slots.png`, `map.png` | первая картинг-версия: вход, заезды, карта трассы (F1 + F2) |
| `theme-login.png`, `theme-slots.png`, `theme-details.png` | гоночная тема (D1) и рекорды трассы (F4) |
| `apex-login.png`, `apex-slots.png`, `apex-track.png` | сентябрь 2026, светлая тема: вход через Telegram / без регистрации, каталог, паспорт трассы (F5) |
| `apex-details-dark.png`, `apex-telegram-dark.png`, `apex-profile-dark.png` | сентябрь 2026, тёмная тема: карточка заезда, вход через Telegram, профиль гостя |

Все снимки — 1170×2532 (iPhone, 390×844 @3x).

Новые (`apex-*`) сняты на живом приложении безголовым Chrome через DevTools Protocol:
эмуляция устройства `390×844`, `deviceScaleFactor: 3`, вход гостем через `POST /auth/demo`,
тема — переключателем в профиле. Персональных данных на снимках нет: профиль — гостевой.

Как снять вручную: открой приложение в Chrome, включи режим устройства (F12 → Ctrl+Shift+M →
iPhone), войди и сделай скриншот (⋮ → Capture screenshot).
