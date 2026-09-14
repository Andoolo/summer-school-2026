package telegramlogin

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"summer-school-2026/backend/internal/service/auth"
	"summer-school-2026/backend/internal/telegram"
)

// memoryRepo — простая реализация Repository в памяти с той же семантикой, что у Postgres.
type memoryRepo struct {
	requests map[string]*memoryRequest // ключ — pollHash
	chats    map[string]int64          // номер → привязанный чат
	notify   map[int64]bool            // чат → уведомления включены
}

type memoryRequest struct {
	startHash string
	Request
	chatID    int64
	createdAt time.Time
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{requests: map[string]*memoryRequest{}, chats: map[string]int64{}, notify: map[int64]bool{}}
}

func (r *memoryRepo) LinkChat(_ context.Context, phone string, chatID int64) error {
	for otherPhone, chat := range r.chats {
		if chat == chatID && otherPhone != phone {
			delete(r.chats, otherPhone)
		}
	}
	r.chats[phone] = chatID
	if _, ok := r.notify[chatID]; !ok {
		r.notify[chatID] = true
	}
	return nil
}

func (r *memoryRepo) SetNotifications(_ context.Context, chatID int64, enabled bool) (bool, error) {
	for _, chat := range r.chats {
		if chat == chatID {
			r.notify[chatID] = enabled
			return true, nil
		}
	}
	return false, nil
}

func (r *memoryRepo) CreateLoginRequest(_ context.Context, startHash, pollHash, code string, now, expiresAt time.Time) error {
	r.requests[pollHash] = &memoryRequest{startHash: startHash, createdAt: now, Request: Request{ConfirmCode: code, Status: "pending", ExpiresAt: expiresAt}}
	return nil
}

func (r *memoryRepo) AttachChat(_ context.Context, startHash string, chatID int64, now time.Time) (Request, bool, error) {
	for _, req := range r.requests {
		if req.startHash == startHash && req.Status == "pending" && now.Before(req.ExpiresAt) && (req.chatID == 0 || req.chatID == chatID) {
			req.chatID = chatID
			req.ChatID = chatID
			return req.Request, true, nil
		}
	}
	return Request{}, false, nil
}

func (r *memoryRepo) ConfirmLatestForChat(_ context.Context, chatID int64, phone, firstName string, now time.Time) (bool, error) {
	var latest *memoryRequest
	for _, req := range r.requests {
		if req.chatID == chatID && req.Status == "pending" && now.Before(req.ExpiresAt) && (latest == nil || req.createdAt.After(latest.createdAt)) {
			latest = req
		}
	}
	if latest == nil {
		return false, nil
	}
	latest.Status, latest.Phone, latest.FirstName = "confirmed", phone, firstName
	return true, nil
}

func (r *memoryRepo) ConsumeConfirmed(_ context.Context, pollHash string, now time.Time) (Request, bool, error) {
	req, ok := r.requests[pollHash]
	if !ok || req.Status != "confirmed" || !now.Before(req.ExpiresAt) {
		return Request{}, false, nil
	}
	req.Status = "consumed"
	return req.Request, true, nil
}

func (r *memoryRepo) RequestByPollHash(_ context.Context, pollHash string) (Request, bool, error) {
	req, ok := r.requests[pollHash]
	if !ok {
		return Request{}, false, nil
	}
	return req.Request, true, nil
}

type fakeSessions struct {
	phones []string
	names  []string
}

func (s *fakeSessions) LoginByVerifiedPhone(_ context.Context, phone, name string) (auth.VerifyCodeResult, error) {
	s.phones = append(s.phones, phone)
	s.names = append(s.names, name)
	return auth.VerifyCodeResult{Token: "session-token", Client: auth.Client{Phone: phone}}, nil
}

type sentMessage struct {
	chatID int64
	text   string
	markup any
}

type fakeBot struct{ sent []sentMessage }

func (b *fakeBot) SendMessage(_ context.Context, chatID int64, text string, markup any) error {
	b.sent = append(b.sent, sentMessage{chatID, text, markup})
	return nil
}

func (b *fakeBot) last(t *testing.T) sentMessage {
	t.Helper()
	if len(b.sent) == 0 {
		t.Fatal("bot sent nothing")
	}
	return b.sent[len(b.sent)-1]
}

type fixture struct {
	service  *Service
	repo     *memoryRepo
	sessions *fakeSessions
	bot      *fakeBot
	now      time.Time
}

func newFixture() *fixture {
	f := &fixture{repo: newMemoryRepo(), sessions: &fakeSessions{}, bot: &fakeBot{}, now: time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)}
	f.service = NewService(f.repo, f.sessions, f.bot, func() string { return "apex_login_bot" }, nil)
	f.service.now = func() time.Time { return f.now }
	return f
}

func startParam(t *testing.T, deepLink string) string {
	t.Helper()
	parsed, err := url.Parse(deepLink)
	if err != nil || parsed.Host != "t.me" || parsed.Path != "/apex_login_bot" {
		t.Fatalf("deep link = %q", deepLink)
	}
	param := parsed.Query().Get("start")
	if !startPattern.MatchString(param) {
		t.Fatalf("start param %q violates Telegram limits", param)
	}
	return param
}

func privateMessage(userID int64, text string, contact *telegram.Contact) telegram.Update {
	return telegram.Update{Message: &telegram.Message{
		From:    &telegram.User{ID: userID, FirstName: "Анна"},
		Chat:    telegram.Chat{ID: userID, Type: "private"},
		Text:    text,
		Contact: contact,
	}}
}

func TestFullLoginFlow(t *testing.T) {
	f := newFixture()
	ctx := context.Background()

	started, err := f.service.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(started.ConfirmCode) != 4 || started.PollToken == "" || !started.ExpiresAt.Equal(f.now.Add(10*time.Minute)) {
		t.Fatalf("Start() = %+v", started)
	}
	param := startParam(t, started.DeepLink)

	if poll, _ := f.service.Poll(ctx, started.PollToken); poll.Status != StatusPending {
		t.Fatalf("status before /start = %s, want pending", poll.Status)
	}

	if err := f.service.HandleUpdate(ctx, privateMessage(777, "/start "+param, nil)); err != nil {
		t.Fatalf("HandleUpdate(/start) error = %v", err)
	}
	reply := f.bot.last(t)
	if !strings.Contains(reply.text, started.ConfirmCode) {
		t.Fatalf("bot must show confirm code, got %q", reply.text)
	}
	keyboard, ok := reply.markup.(telegram.ReplyKeyboard)
	if !ok || !keyboard.Keyboard[0][0].RequestContact {
		t.Fatalf("bot must offer share-contact button, got %#v", reply.markup)
	}

	contact := &telegram.Contact{PhoneNumber: "79991234567", FirstName: "Анна", UserID: 777}
	if err := f.service.HandleUpdate(ctx, privateMessage(777, "", contact)); err != nil {
		t.Fatalf("HandleUpdate(contact) error = %v", err)
	}
	if !strings.Contains(f.bot.last(t).text, "Готово") {
		t.Fatalf("bot reply = %q", f.bot.last(t).text)
	}

	poll, err := f.service.Poll(ctx, started.PollToken)
	if err != nil || poll.Status != StatusConfirmed || poll.Session.Token != "session-token" {
		t.Fatalf("Poll() = %+v, %v; want confirmed with session", poll, err)
	}
	if len(f.sessions.phones) != 1 || f.sessions.phones[0] != "+79991234567" || f.sessions.names[0] != "Анна" {
		t.Fatalf("session issued for %v / %v", f.sessions.phones, f.sessions.names)
	}

	// Повторный опрос тем же секретом сессию второй раз не выдаёт.
	if again, _ := f.service.Poll(ctx, started.PollToken); again.Status != StatusExpired || len(f.sessions.phones) != 1 {
		t.Fatalf("second poll = %s, sessions = %d; want expired and no new session", again.Status, len(f.sessions.phones))
	}
}

func TestForeignContactIsRejected(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	started, _ := f.service.Start(ctx)
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/start "+startParam(t, started.DeepLink), nil))

	// Пересланный чужой контакт: user_id не совпадает с отправителем.
	foreign := &telegram.Contact{PhoneNumber: "79990000001", FirstName: "Чужой", UserID: 999}
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "", foreign))
	// Контакт без user_id (записан вручную) — тоже не подтверждённый номер.
	manual := &telegram.Contact{PhoneNumber: "79990000002", FirstName: "Вручную"}
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "", manual))

	if poll, _ := f.service.Poll(ctx, started.PollToken); poll.Status != StatusPending {
		t.Fatalf("status = %s, want pending: foreign contact must not confirm", poll.Status)
	}
	if len(f.sessions.phones) != 0 {
		t.Fatal("no session must be issued for foreign contact")
	}
}

func TestContactWithoutStartDoesNothing(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	started, _ := f.service.Start(ctx)

	// Человек не открывал ссылку этого запроса — его номер не должен подтвердить чужой запрос.
	_ = f.service.HandleUpdate(ctx, privateMessage(555, "", &telegram.Contact{PhoneNumber: "79991112233", UserID: 555}))
	if poll, _ := f.service.Poll(ctx, started.PollToken); poll.Status != StatusPending {
		t.Fatalf("status = %s, want pending", poll.Status)
	}
}

func TestLinkCannotBeReusedByAnotherChat(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	started, _ := f.service.Start(ctx)
	param := startParam(t, started.DeepLink)

	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/start "+param, nil))
	_ = f.service.HandleUpdate(ctx, privateMessage(888, "/start "+param, nil))
	if !strings.Contains(f.bot.last(t).text, "устарела") {
		t.Fatalf("second chat must be refused, got %q", f.bot.last(t).text)
	}
}

func TestExpiredRequest(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	started, _ := f.service.Start(ctx)
	f.now = f.now.Add(11 * time.Minute)

	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/start "+startParam(t, started.DeepLink), nil))
	if !strings.Contains(f.bot.last(t).text, "устарела") {
		t.Fatalf("expired link reply = %q", f.bot.last(t).text)
	}
	if poll, _ := f.service.Poll(ctx, started.PollToken); poll.Status != StatusExpired {
		t.Fatalf("status = %s, want expired", poll.Status)
	}
}

func TestIgnoresGroupsBotsAndGarbage(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	group := privateMessage(1, "/start abc", nil)
	group.Message.Chat.Type = "group"
	bot := privateMessage(2, "/start abc", nil)
	bot.Message.From.IsBot = true
	for _, update := range []telegram.Update{group, bot, {}} {
		if err := f.service.HandleUpdate(ctx, update); err != nil {
			t.Fatalf("HandleUpdate() error = %v", err)
		}
	}
	if len(f.bot.sent) != 0 {
		t.Fatalf("bot must stay silent in groups and for bots, sent %d", len(f.bot.sent))
	}

	_ = f.service.HandleUpdate(ctx, privateMessage(3, "/start", nil))
	_ = f.service.HandleUpdate(ctx, privateMessage(3, "/start ../../etc", nil))
	_ = f.service.HandleUpdate(ctx, privateMessage(3, "привет", nil))
	for _, msg := range f.bot.sent {
		if !strings.Contains(msg.text, "бот приложения «Апекс»") {
			t.Fatalf("unexpected reply %q", msg.text)
		}
	}
}

func TestStartRequiresConfiguredBot(t *testing.T) {
	f := newFixture()
	f.service.username = func() string { return "" }
	if _, err := f.service.Start(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Start() error = %v, want %v", err, ErrNotConfigured)
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"79991234567":   "+79991234567",
		"+79991234567":  "+79991234567",
		"+7 999 123-45": "+799912345",
		"447700900123":  "+447700900123",
	}
	for raw, want := range cases {
		if got, ok := NormalizePhone(raw); !ok || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", raw, got, ok, want)
		}
	}
	for _, bad := range []string{"", "abc", "0123", "70001234567", "+7000 123 45 67"} {
		if _, ok := NormalizePhone(bad); ok {
			t.Errorf("NormalizePhone(%q) must be rejected", bad)
		}
	}
}

func TestRandomCodeUsesUnambiguousAlphabet(t *testing.T) {
	for i := 0; i < 200; i++ {
		code, err := randomCode()
		if err != nil || len(code) != 4 {
			t.Fatalf("randomCode() = %q, %v", code, err)
		}
		for _, r := range code {
			if !strings.ContainsRune(codeAlphabet, r) {
				t.Fatalf("code %q has char outside alphabet", code)
			}
		}
	}
}

func TestCodeTypedIntoBotGetsHint(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	started, _ := f.service.Start(ctx)
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/start "+startParam(t, started.DeepLink), nil))

	// Латиница, кириллица и пробелы по краям — всё это код, набранный вручную.
	for _, typed := range []string{started.ConfirmCode, "T73X", " nt4p ", "Т73Х"} {
		_ = f.service.HandleUpdate(ctx, privateMessage(777, typed, nil))
		if reply := f.bot.last(t); !strings.Contains(reply.text, "вводить не нужно") {
			t.Fatalf("reply to %q = %q, want code hint", typed, reply.text)
		}
	}
	// Код текстом не подтверждает вход — подтверждает только номер.
	if poll, _ := f.service.Poll(ctx, started.PollToken); poll.Status != StatusPending {
		t.Fatalf("status = %s, want pending: typed code must not confirm", poll.Status)
	}

	for _, other := range []string{"привет", "12345", "ok", "как войти?"} {
		_ = f.service.HandleUpdate(ctx, privateMessage(777, other, nil))
		if reply := f.bot.last(t); !strings.Contains(reply.text, "бот приложения «Апекс»") {
			t.Fatalf("reply to %q = %q, want general help", other, reply.text)
		}
	}
}

func TestLoginLinksChatForNotifications(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	started, _ := f.service.Start(ctx)
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/start "+startParam(t, started.DeepLink), nil))
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "", &telegram.Contact{PhoneNumber: "79991234567", UserID: 777}))
	if !strings.Contains(f.bot.last(t).text, "/stop") {
		t.Fatalf("done message must mention notifications, got %q", f.bot.last(t).text)
	}

	// Чат привязывается только когда приложение забрало сессию, а не при подтверждении в боте.
	if len(f.repo.chats) != 0 {
		t.Fatalf("chat linked before poll: %v", f.repo.chats)
	}
	if poll, _ := f.service.Poll(ctx, started.PollToken); poll.Status != StatusConfirmed {
		t.Fatalf("status = %s, want confirmed", poll.Status)
	}
	if f.repo.chats["+79991234567"] != 777 {
		t.Fatalf("chats = %v, want +79991234567 → 777", f.repo.chats)
	}
}

func TestStopAndNotifyCommands(t *testing.T) {
	f := newFixture()
	ctx := context.Background()

	// Чат, не связанный с аккаунтом, получает объяснение.
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/stop", nil))
	if !strings.Contains(f.bot.last(t).text, "не связан") {
		t.Fatalf("unlinked /stop reply = %q", f.bot.last(t).text)
	}

	_ = f.repo.LinkChat(ctx, "+79991234567", 777)
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/stop", nil))
	if f.repo.notify[777] || !strings.Contains(f.bot.last(t).text, "отключены") {
		t.Fatalf("after /stop notify=%v reply=%q", f.repo.notify[777], f.bot.last(t).text)
	}
	// Команда из меню приходит с именем бота.
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/notify@apex_login_bot", nil))
	if !f.repo.notify[777] || !strings.Contains(f.bot.last(t).text, "включены") {
		t.Fatalf("after /notify notify=%v reply=%q", f.repo.notify[777], f.bot.last(t).text)
	}
	// Похожий текст — не команда.
	_ = f.service.HandleUpdate(ctx, privateMessage(777, "/stopall", nil))
	if !f.repo.notify[777] {
		t.Fatal("/stopall must not disable notifications")
	}
}
