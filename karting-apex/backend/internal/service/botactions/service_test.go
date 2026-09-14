package botactions

import (
	"context"
	"errors"
	"testing"
	"time"

	"summer-school-2026/backend/internal/service/booking"
	"summer-school-2026/backend/internal/telegram"
)

const (
	bookingID = "11111111-1111-1111-1111-111111111111"
	chatID    = int64(777)
)

var now = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

type fakeRepo struct {
	booking   Booking
	found     bool
	cancelErr error
	lookups   []int64
	cancelled []string
}

func (r *fakeRepo) BookingForChat(_ context.Context, id string, chat int64) (Booking, bool, error) {
	r.lookups = append(r.lookups, chat)
	if id != bookingID {
		return Booking{}, false, nil
	}
	return r.booking, r.found, nil
}

func (r *fakeRepo) Cancel(_ context.Context, clientID, id string, _ time.Time) (booking.Booking, error) {
	if r.cancelErr != nil {
		return booking.Booking{}, r.cancelErr
	}
	r.cancelled = append(r.cancelled, clientID+"/"+id)
	return booking.Booking{ID: id, Status: "cancelled"}, nil
}

type edit struct {
	messageID int64
	markup    telegram.InlineKeyboard
}

type fakeBot struct {
	answers []string
	edits   []edit
}

func (b *fakeBot) AnswerCallbackQuery(_ context.Context, _ string, text string) error {
	b.answers = append(b.answers, text)
	return nil
}

func (b *fakeBot) EditMessageReplyMarkup(_ context.Context, _, messageID int64, markup telegram.InlineKeyboard) error {
	b.edits = append(b.edits, edit{messageID, markup})
	return nil
}

func press(data string) telegram.CallbackQuery {
	return telegram.CallbackQuery{
		ID:      "cb",
		From:    telegram.User{ID: chatID},
		Message: &telegram.Message{MessageID: 55, Chat: telegram.Chat{ID: chatID, Type: "private"}},
		Data:    data,
	}
}

func newService(repo *fakeRepo, bot *fakeBot, changes *int) *Service {
	s := NewService(repo, bot, func() { *changes++ }, nil)
	s.now = func() time.Time { return now }
	return s
}

func activeBooking(startIn time.Duration) *fakeRepo {
	return &fakeRepo{booking: Booking{ClientID: "client-1", Status: "active", StartAt: now.Add(startIn)}, found: true}
}

func buttons(k telegram.InlineKeyboard) []string {
	var data []string
	for _, row := range k.InlineKeyboard {
		for _, button := range row {
			data = append(data, button.CallbackData)
		}
	}
	return data
}

func TestCancelFlowAsksThenCancels(t *testing.T) {
	repo, bot, changes := activeBooking(5*time.Hour), &fakeBot{}, 0
	s := newService(repo, bot, &changes)
	ctx := context.Background()

	// 1. «Отменить бронь» — только спрашиваем.
	if err := s.HandleCallback(ctx, press(CancelKeyboard(bookingID).InlineKeyboard[0][0].CallbackData)); err != nil {
		t.Fatal(err)
	}
	if len(repo.cancelled) != 0 || bot.answers[0] != textAskConfirm {
		t.Fatalf("first press cancelled=%v answer=%q", repo.cancelled, bot.answers)
	}
	if got := buttons(bot.edits[0].markup); len(got) != 2 || got[0] != actionConfirm+bookingID || got[1] != actionKeep+bookingID {
		t.Fatalf("confirm buttons = %v", got)
	}

	// 2. «Нет, оставить» — кнопка возвращается.
	_ = s.HandleCallback(ctx, press(actionKeep+bookingID))
	if len(repo.cancelled) != 0 || bot.answers[1] != textKept || buttons(bot.edits[1].markup)[0] != actionCancel+bookingID {
		t.Fatalf("keep: cancelled=%v answers=%v", repo.cancelled, bot.answers)
	}

	// 3. «Да, отменить» — отменяем, кнопки убираем, будим рассыльщика.
	_ = s.HandleCallback(ctx, press(actionConfirm+bookingID))
	if len(repo.cancelled) != 1 || repo.cancelled[0] != "client-1/"+bookingID {
		t.Fatalf("cancelled = %v", repo.cancelled)
	}
	if bot.answers[2] != textCancelled || len(bot.edits[2].markup.InlineKeyboard) != 0 || changes != 1 {
		t.Fatalf("after confirm answers=%v edits=%v changes=%d", bot.answers, bot.edits, changes)
	}
}

func TestLateCancelWarns(t *testing.T) {
	repo, bot, changes := activeBooking(90*time.Minute), &fakeBot{}, 0
	_ = newService(repo, bot, &changes).HandleCallback(context.Background(), press(actionCancel+bookingID))
	if bot.answers[0] != textAskLate {
		t.Fatalf("answer = %q, want late cancel warning", bot.answers[0])
	}
}

func TestOnlyOwnChatAndValidData(t *testing.T) {
	ctx := context.Background()

	// Нажатие переслано или подделано: отправитель не совпадает с чатом.
	repo, bot, changes := activeBooking(5*time.Hour), &fakeBot{}, 0
	forged := press(actionConfirm + bookingID)
	forged.From.ID = 999
	_ = newService(repo, bot, &changes).HandleCallback(ctx, forged)
	if len(repo.lookups) != 0 || len(repo.cancelled) != 0 || len(bot.answers) != 1 {
		t.Fatalf("forged press: lookups=%v cancelled=%v answers=%v", repo.lookups, repo.cancelled, bot.answers)
	}

	// Мусор в данных кнопки — в базу не ходим, но на нажатие отвечаем.
	for _, data := range []string{"", "cancel:", "cancel:not-a-uuid", "drop:" + bookingID, "cancel_yes:1'; DROP TABLE bookings;--"} {
		repo, bot := activeBooking(5*time.Hour), &fakeBot{}
		_ = newService(repo, bot, &changes).HandleCallback(ctx, press(data))
		if len(repo.lookups) != 0 || len(bot.answers) != 1 {
			t.Fatalf("data %q: lookups=%v answers=%v", data, repo.lookups, bot.answers)
		}
	}

	// Чужая бронь (не найдена среди броней клиента этого чата).
	repo, bot = &fakeRepo{}, &fakeBot{}
	_ = newService(repo, bot, &changes).HandleCallback(ctx, press(actionConfirm+bookingID))
	if bot.answers[0] != textNotFound || len(repo.cancelled) != 0 || repo.lookups[0] != chatID {
		t.Fatalf("foreign booking answers=%v cancelled=%v", bot.answers, repo.cancelled)
	}
}

func TestStaleButtons(t *testing.T) {
	ctx := context.Background()
	changes := 0
	cases := []struct {
		name string
		repo *fakeRepo
		want string
	}{
		{"cancelled in app", &fakeRepo{booking: Booking{Status: "cancelled", StartAt: now.Add(time.Hour)}, found: true}, textAlready},
		{"race started", activeBooking(-time.Minute), textStarted},
		{"cancelled concurrently", &fakeRepo{booking: Booking{ClientID: "c", Status: "active", StartAt: now.Add(5 * time.Hour)}, found: true, cancelErr: booking.ErrAlreadyCancelled}, textAlready},
	}
	for _, tc := range cases {
		bot := &fakeBot{}
		_ = newService(tc.repo, bot, &changes).HandleCallback(ctx, press(actionConfirm+bookingID))
		if bot.answers[0] != tc.want || len(bot.edits) != 1 || len(bot.edits[0].markup.InlineKeyboard) != 0 {
			t.Fatalf("%s: answers=%v edits=%v", tc.name, bot.answers, bot.edits)
		}
	}
	if changes != 0 {
		t.Fatalf("nothing was cancelled, but dispatcher woken %d times", changes)
	}

	// Сбой базы — кнопки оставляем, чтобы можно было нажать ещё раз.
	repo := &fakeRepo{booking: Booking{ClientID: "c", Status: "active", StartAt: now.Add(5 * time.Hour)}, found: true, cancelErr: errors.New("db down")}
	bot := &fakeBot{}
	if err := newService(repo, bot, &changes).HandleCallback(ctx, press(actionConfirm+bookingID)); err == nil {
		t.Fatal("database error must be returned for logging")
	}
	if bot.answers[0] != textCancelFailed || len(bot.edits) != 0 {
		t.Fatalf("db error answers=%v edits=%v", bot.answers, bot.edits)
	}
}

func TestCallbackDataFitsTelegramLimit(t *testing.T) {
	for _, data := range append(buttons(CancelKeyboard(bookingID)), buttons(confirmKeyboard(bookingID))...) {
		if len(data) > 64 {
			t.Fatalf("callback data %q is %d bytes, Telegram allows 64", data, len(data))
		}
	}
}
