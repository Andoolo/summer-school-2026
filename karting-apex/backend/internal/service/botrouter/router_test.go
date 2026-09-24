package botrouter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"summer-school-2026/backend/internal/telegram"
)

type fakeLogin struct {
	handles map[string]bool // текст сообщения → вход его обработал
	seen    []string
	err     error
}

func (l *fakeLogin) HandleMessage(_ context.Context, msg *telegram.Message) (bool, error) {
	l.seen = append(l.seen, msg.Text)
	return l.handles[msg.Text], l.err
}

type fakeCallbacks struct{ data []string }

func (c *fakeCallbacks) HandleCallback(_ context.Context, q telegram.CallbackQuery) error {
	c.data = append(c.data, q.Data)
	return nil
}

type fakeNotifications struct {
	linked  map[int64]bool
	enabled map[int64]bool
}

func (n *fakeNotifications) SetNotifications(_ context.Context, chatID int64, enabled bool) (bool, error) {
	if !n.linked[chatID] {
		return false, nil
	}
	n.enabled[chatID] = enabled
	return true, nil
}

type sent struct {
	chatID int64
	text   string
}

type fakeBot struct{ sent []sent }

func (b *fakeBot) SendMessage(_ context.Context, chatID int64, text string, _ any) error {
	b.sent = append(b.sent, sent{chatID, text})
	return nil
}

func (b *fakeBot) last(t *testing.T) string {
	t.Helper()
	if len(b.sent) == 0 {
		t.Fatal("bot sent nothing")
	}
	return b.sent[len(b.sent)-1].text
}

type fixture struct {
	router        *Router
	login         *fakeLogin
	callbacks     *fakeCallbacks
	notifications *fakeNotifications
	bot           *fakeBot
}

func newFixture(cfg Config) *fixture {
	f := &fixture{
		login:         &fakeLogin{handles: map[string]bool{}},
		callbacks:     &fakeCallbacks{},
		notifications: &fakeNotifications{linked: map[int64]bool{}, enabled: map[int64]bool{}},
		bot:           &fakeBot{},
	}
	cfg.Login, cfg.Notifications, cfg.Bot = f.login, f.notifications, f.bot
	if cfg.Callbacks == nil {
		cfg.Callbacks = f.callbacks
	}
	f.router = New(cfg)
	return f
}

func private(chatID int64, text string) telegram.Update {
	return telegram.Update{Message: &telegram.Message{
		From: &telegram.User{ID: chatID, FirstName: "Анна"},
		Chat: telegram.Chat{ID: chatID, Type: "private"},
		Text: text,
	}}
}

func TestLoginGoesFirstThenCommandsThenHelp(t *testing.T) {
	f := newFixture(Config{})
	f.login.handles["/start token"] = true
	ctx := context.Background()

	if err := f.router.HandleUpdate(ctx, private(7, "/start token")); err != nil {
		t.Fatal(err)
	}
	if len(f.bot.sent) != 0 {
		t.Fatalf("message handled by login must not get a router reply, sent %v", f.bot.sent)
	}

	for _, text := range []string{"привет", "/start", "12345"} {
		_ = f.router.HandleUpdate(ctx, private(7, text))
		if !strings.Contains(f.bot.last(t), "бот приложения «Апекс»") {
			t.Fatalf("reply to %q = %q, want help", text, f.bot.last(t))
		}
	}
	if strings.Join(f.login.seen, "|") != "/start token|привет|/start|12345" {
		t.Fatalf("login must see every private message first, saw %v", f.login.seen)
	}

	f.login.err = errors.New("db down")
	if err := f.router.HandleUpdate(ctx, private(7, "что угодно")); err == nil {
		t.Fatal("login error must be returned for logging")
	}
}

func TestIgnoresGroupsBotsAndEmptyUpdates(t *testing.T) {
	f := newFixture(Config{})
	group := private(1, "/stop")
	group.Message.Chat.Type = "group"
	fromBot := private(2, "/stop")
	fromBot.Message.From.IsBot = true
	noSender := private(3, "/stop")
	noSender.Message.From = nil
	for _, update := range []telegram.Update{group, fromBot, noSender, {}} {
		if err := f.router.HandleUpdate(context.Background(), update); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.bot.sent) != 0 || len(f.login.seen) != 0 {
		t.Fatalf("bot must stay silent: sent %v, login saw %v", f.bot.sent, f.login.seen)
	}
}

func TestStopAndNotifyCommands(t *testing.T) {
	f := newFixture(Config{})
	ctx := context.Background()

	_ = f.router.HandleUpdate(ctx, private(777, "/stop"))
	if !strings.Contains(f.bot.last(t), "не связан") {
		t.Fatalf("unlinked /stop reply = %q", f.bot.last(t))
	}

	f.notifications.linked[777] = true
	_ = f.router.HandleUpdate(ctx, private(777, "/stop"))
	if f.notifications.enabled[777] || !strings.Contains(f.bot.last(t), "отключены") {
		t.Fatalf("after /stop enabled=%v reply=%q", f.notifications.enabled[777], f.bot.last(t))
	}
	// Команда из меню приходит с именем бота.
	_ = f.router.HandleUpdate(ctx, private(777, "/notify@apex_karting_bot"))
	if !f.notifications.enabled[777] || !strings.Contains(f.bot.last(t), "включены") {
		t.Fatalf("after /notify enabled=%v reply=%q", f.notifications.enabled[777], f.bot.last(t))
	}
	// Похожий текст — не команда.
	_ = f.router.HandleUpdate(ctx, private(777, "/stopall"))
	if !f.notifications.enabled[777] || !strings.Contains(f.bot.last(t), "бот приложения") {
		t.Fatal("/stopall must not disable notifications")
	}
}

func TestCallbacks(t *testing.T) {
	f := newFixture(Config{})
	press := telegram.Update{CallbackQuery: &telegram.CallbackQuery{ID: "cb", Data: "cancel:x"}}
	if err := f.router.HandleUpdate(context.Background(), press); err != nil {
		t.Fatal(err)
	}
	if strings.Join(f.callbacks.data, ",") != "cancel:x" || len(f.login.seen) != 0 || len(f.bot.sent) != 0 {
		t.Fatalf("callback routed to %v, login saw %v, sent %v", f.callbacks.data, f.login.seen, f.bot.sent)
	}

	// Без обработчика кнопок нажатия просто игнорируются.
	noButtons := New(Config{Login: &fakeLogin{}, Notifications: &fakeNotifications{}, Bot: &fakeBot{}})
	if err := noButtons.HandleUpdate(context.Background(), press); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCommands(t *testing.T) {
	calls := 0
	stats := func(context.Context) (string, error) {
		calls++
		return "📊 сводка", nil
	}
	ctx := context.Background()

	withoutAdmin := newFixture(Config{Stats: stats})
	_ = withoutAdmin.router.HandleUpdate(ctx, private(555, "/whoami"))
	if withoutAdmin.bot.last(t) != "Ваш Telegram chat id: 555" {
		t.Fatalf("/whoami reply = %q", withoutAdmin.bot.last(t))
	}
	// Администратор не задан — /stats как неизвестная команда.
	_ = withoutAdmin.router.HandleUpdate(ctx, private(555, "/stats"))
	if calls != 0 || !strings.Contains(withoutAdmin.bot.last(t), "бот приложения «Апекс»") {
		t.Fatalf("/stats without admin: calls=%d reply=%q", calls, withoutAdmin.bot.last(t))
	}

	f := newFixture(Config{AdminChatID: 555, Stats: stats})
	_ = f.router.HandleUpdate(ctx, private(777, "/stats"))
	if calls != 0 || !strings.Contains(f.bot.last(t), "бот приложения «Апекс»") {
		t.Fatalf("non-admin /stats: calls=%d reply=%q", calls, f.bot.last(t))
	}
	_ = f.router.HandleUpdate(ctx, private(555, "/stats"))
	if calls != 1 || f.bot.last(t) != "📊 сводка" {
		t.Fatalf("admin /stats: calls=%d reply=%q", calls, f.bot.last(t))
	}

	failing := newFixture(Config{AdminChatID: 555, Stats: func(context.Context) (string, error) { return "", errors.New("db down") }})
	if err := failing.router.HandleUpdate(ctx, private(555, "/stats")); err == nil {
		t.Fatal("stats error must be returned for logging")
	}
	if !strings.Contains(failing.bot.last(t), "Не удалось собрать сводку") {
		t.Fatalf("stats failure reply = %q", failing.bot.last(t))
	}
}
