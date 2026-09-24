// Package botrouter разбирает входящие обновления бота. Вебхук у бота один на всё:
// вход через Telegram, нажатия кнопок под уведомлениями и команды. Каждое действие живёт в
// своём сервисе, а здесь только решается, кому отдать обновление.
//
// Порядок: нажатие кнопки → Callbacks; сообщение → сначала вход (ссылка /start, контакт,
// код сверки), затем команды (/stop, /notify, /whoami, /stats), иначе — справка.
package botrouter

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"summer-school-2026/backend/internal/telegram"
)

// Login — вход через Telegram. handled=false — сообщение не относится ко входу.
type Login interface {
	HandleMessage(ctx context.Context, msg *telegram.Message) (handled bool, err error)
}

// Callbacks — нажатия кнопок под сообщениями бота (например, «Отменить бронь»).
type Callbacks interface {
	HandleCallback(ctx context.Context, q telegram.CallbackQuery) error
}

// NotificationSettings — включение и выключение уведомлений о бронях для чата. found=false —
// чат не привязан ни к одному клиенту.
type NotificationSettings interface {
	SetNotifications(ctx context.Context, chatID int64, enabled bool) (found bool, err error)
}

type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string, markup any) error
}

// Config — всё, что нужно роутеру. Необязательные части: Callbacks (nil — нажатия
// игнорируются) и Stats с AdminChatID (0 — /stats выключен, отвечает справкой).
type Config struct {
	Login         Login
	Callbacks     Callbacks
	Notifications NotificationSettings
	Bot           Sender
	AdminChatID   int64
	Stats         func(context.Context) (string, error)
	Logger        *slog.Logger
}

type Router struct {
	login         Login
	callbacks     Callbacks
	notifications NotificationSettings
	bot           Sender
	adminChatID   int64
	stats         func(context.Context) (string, error)
	logger        *slog.Logger
}

func New(cfg Config) *Router {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Router{
		login:         cfg.Login,
		callbacks:     cfg.Callbacks,
		notifications: cfg.Notifications,
		bot:           cfg.Bot,
		adminChatID:   cfg.AdminChatID,
		stats:         cfg.Stats,
		logger:        cfg.Logger,
	}
}

const (
	textHelp = "Это бот приложения «Апекс»: вход без кода и уведомления о бронях.\n\n" +
		"Откройте приложение и нажмите «Войти через Telegram» — бот пришлёт кнопку для входа. " +
		"После входа сюда будут приходить подтверждения броней и напоминания о заездах.\n\n" +
		"/stop — отключить уведомления\n/notify — включить снова"
	textNotifyOff    = "Уведомления о бронях отключены. Включить снова: /notify"
	textNotifyOn     = "Уведомления о бронях включены. Отключить: /stop"
	textNotifyNoChat = "Этот чат пока не связан с аккаунтом «Апекса».\n\n" +
		"Войдите в приложение через Telegram — и бот будет присылать уведомления о бронях."
	textStatsFailed = "Не удалось собрать сводку: база недоступна. Подробности — в журнале сервиса."
)

// HandleUpdate обрабатывает обновление из вебхука. Ошибки отправки сообщений только
// логируются: вебхук должен ответить Telegram быстро, иначе тот начнёт повторы.
func (r *Router) HandleUpdate(ctx context.Context, update telegram.Update) error {
	if update.CallbackQuery != nil {
		if r.callbacks == nil {
			return nil
		}
		return r.callbacks.HandleCallback(ctx, *update.CallbackQuery)
	}
	msg := update.Message
	// Бот работает только в личной переписке с человеком: в группах и от других ботов молчит.
	if msg == nil || msg.From == nil || msg.From.IsBot || msg.Chat.Type != "private" {
		return nil
	}
	if handled, err := r.login.HandleMessage(ctx, msg); handled || err != nil {
		return err
	}

	chatID := msg.Chat.ID
	switch {
	case isCommand(msg.Text, "/stop"):
		return r.setNotifications(ctx, chatID, false)
	case isCommand(msg.Text, "/notify"):
		return r.setNotifications(ctx, chatID, true)
	case isCommand(msg.Text, "/whoami"):
		r.send(ctx, chatID, "Ваш Telegram chat id: "+strconv.FormatInt(chatID, 10))
		return nil
	case isCommand(msg.Text, "/stats") && r.isAdmin(chatID):
		text, err := r.stats(ctx)
		if err != nil {
			r.send(ctx, chatID, textStatsFailed)
			return err
		}
		r.send(ctx, chatID, text)
		return nil
	default:
		r.send(ctx, chatID, textHelp)
		return nil
	}
}

// isAdmin — чат администратора, и сводка подключена. Остальным /stats не виден: они
// получают обычную справку.
func (r *Router) isAdmin(chatID int64) bool {
	return r.stats != nil && r.adminChatID != 0 && chatID == r.adminChatID
}

func (r *Router) setNotifications(ctx context.Context, chatID int64, enabled bool) error {
	found, err := r.notifications.SetNotifications(ctx, chatID, enabled)
	if err != nil {
		return err
	}
	switch {
	case !found:
		r.send(ctx, chatID, textNotifyNoChat)
	case enabled:
		r.send(ctx, chatID, textNotifyOn)
	default:
		r.send(ctx, chatID, textNotifyOff)
	}
	return nil
}

func (r *Router) send(ctx context.Context, chatID int64, text string) {
	if err := r.bot.SendMessage(ctx, chatID, text, nil); err != nil {
		r.logger.Error("telegram send message failed", "error", err)
	}
}

// isCommand — текст равен команде, в том числе в виде /stop@имя_бота из меню команд.
func isCommand(text, command string) bool {
	text = strings.TrimSpace(text)
	return text == command || strings.HasPrefix(text, command+"@")
}
