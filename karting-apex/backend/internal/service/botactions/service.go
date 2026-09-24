// Package botactions — действия по кнопкам под сообщениями бота. Пока одно: отмена брони.
//
// Порядок: «Отменить бронь» → бот меняет кнопки на «Да, отменить / Нет, оставить» →
// «Да» отменяет бронь. Второй шаг защищает от случайного нажатия. Сообщение об отмене
// приходит обычным уведомлением, освободившееся место уходит листу ожидания.
//
// Кто нажал, подтверждает Telegram (CallbackQuery.From). Данные кнопки приходят от клиента
// и могут быть подделаны, поэтому бронь ищется только среди броней клиента, к которому
// привязан этот чат.
package botactions

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"summer-school-2026/backend/internal/service/booking"
	"summer-school-2026/backend/internal/telegram"

	"github.com/google/uuid"
)

const (
	actionCancel  = "cancel:"
	actionConfirm = "cancel_yes:"
	actionKeep    = "cancel_no:"
)

// CancelKeyboard — кнопка «Отменить бронь» под уведомлением.
func CancelKeyboard(bookingID string) telegram.InlineKeyboard {
	return telegram.InlineKeyboard{InlineKeyboard: [][]telegram.InlineButton{
		{{Text: "Отменить бронь", CallbackData: actionCancel + bookingID}},
	}}
}

func confirmKeyboard(bookingID string) telegram.InlineKeyboard {
	return telegram.InlineKeyboard{InlineKeyboard: [][]telegram.InlineButton{
		{{Text: "Да, отменить", CallbackData: actionConfirm + bookingID}},
		{{Text: "Нет, оставить", CallbackData: actionKeep + bookingID}},
	}}
}

// Booking — то, что нужно знать о брони для отмены из бота.
type Booking struct {
	ClientID string
	Status   booking.Status
	StartAt  time.Time
}

// Finder ищет бронь среди броней клиента, к которому привязан чат.
type Finder interface {
	BookingForChat(ctx context.Context, bookingID string, chatID int64) (Booking, bool, error)
}

// Canceller — отмена брони, та же, что в приложении (booking.Repository).
type Canceller interface {
	Cancel(ctx context.Context, clientID, bookingID string, now time.Time) (booking.Booking, error)
}

type Bot interface {
	AnswerCallbackQuery(ctx context.Context, callbackQueryID, text string, showAlert bool) error
	EditMessageReplyMarkup(ctx context.Context, chatID, messageID int64, markup telegram.InlineKeyboard) error
}

type Service struct {
	finder    Finder
	canceller Canceller
	bot       Bot
	logger    *slog.Logger
	now       func() time.Time
	onChange  func()
}

// NewService: onChange — сигнал рассыльщику после отмены (сообщение об отмене и лист ожидания).
func NewService(finder Finder, canceller Canceller, bot Bot, onChange func(), logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if onChange == nil {
		onChange = func() {}
	}
	return &Service{finder: finder, canceller: canceller, bot: bot, logger: logger, now: time.Now, onChange: onChange}
}

const (
	textAskConfirm   = "Точно отменить бронь?"
	textAskLate      = "До старта меньше 2 часов: это поздняя отмена, место за вами не освободится. Отменить?"
	textCancelled    = "Бронь отменена"
	textKept         = "Бронь сохранена"
	textAlready      = "Бронь уже отменена"
	textStarted      = "Заезд уже начался — отменить нельзя"
	textNotFound     = "Бронь не найдена"
	textCancelFailed = "Не получилось отменить. Попробуйте ещё раз или отмените в приложении."
)

// HandleCallback обрабатывает нажатие кнопки. На нажатие всегда отвечаем — иначе на кнопке
// крутятся «часики».
func (s *Service) HandleCallback(ctx context.Context, q telegram.CallbackQuery) error {
	action, bookingID, ok := parse(q.Data)
	// Кнопки есть только в личном чате с ботом; там id чата совпадает с id человека.
	if !ok || q.Message == nil || q.Message.Chat.Type != "private" || q.Message.Chat.ID != q.From.ID || q.From.IsBot {
		s.answer(ctx, q, "")
		return nil
	}
	chatID, messageID := q.Message.Chat.ID, q.Message.MessageID

	found, exists, err := s.finder.BookingForChat(ctx, bookingID, chatID)
	if err != nil {
		s.answer(ctx, q, textCancelFailed)
		return err
	}
	now := s.now().UTC()
	switch {
	case !exists:
		s.finish(ctx, q, chatID, messageID, textNotFound)
		return nil
	case found.Status != booking.StatusActive:
		s.finish(ctx, q, chatID, messageID, textAlready)
		return nil
	case !now.Before(found.StartAt):
		s.finish(ctx, q, chatID, messageID, textStarted)
		return nil
	}

	switch action {
	case actionCancel:
		s.edit(ctx, chatID, messageID, confirmKeyboard(bookingID))
		if status, _ := booking.CancellationStatus(now, found.StartAt); status == booking.StatusLateCancel {
			// Поздняя отмена не освобождает место — это окно, которое нужно закрыть, а не
			// исчезающая подсказка.
			s.alert(ctx, q, textAskLate)
		} else {
			s.answer(ctx, q, textAskConfirm)
		}
	case actionKeep:
		s.edit(ctx, chatID, messageID, CancelKeyboard(bookingID))
		s.answer(ctx, q, textKept)
	case actionConfirm:
		_, err := s.canceller.Cancel(ctx, found.ClientID, bookingID, now)
		switch {
		case err == nil:
			s.logger.Info("booking cancelled from telegram", "booking_id", bookingID)
			s.onChange()
			s.finish(ctx, q, chatID, messageID, textCancelled)
		case errors.Is(err, booking.ErrAlreadyCancelled):
			s.finish(ctx, q, chatID, messageID, textAlready)
		case errors.Is(err, booking.ErrSlotStarted):
			s.finish(ctx, q, chatID, messageID, textStarted)
		case errors.Is(err, booking.ErrNotFound), errors.Is(err, booking.ErrForbidden):
			s.finish(ctx, q, chatID, messageID, textNotFound)
		default:
			s.answer(ctx, q, textCancelFailed)
			return err
		}
	}
	return nil
}

// finish убирает кнопки (с брони больше нечего делать) и показывает итог.
func (s *Service) finish(ctx context.Context, q telegram.CallbackQuery, chatID, messageID int64, text string) {
	s.edit(ctx, chatID, messageID, telegram.InlineKeyboard{})
	s.answer(ctx, q, text)
}

func (s *Service) edit(ctx context.Context, chatID, messageID int64, markup telegram.InlineKeyboard) {
	if err := s.bot.EditMessageReplyMarkup(ctx, chatID, messageID, markup); err != nil {
		// «message is not modified» при повторном нажатии — не ошибка для человека.
		s.logger.Warn("telegram edit reply markup failed", "error", err)
	}
}

func (s *Service) answer(ctx context.Context, q telegram.CallbackQuery, text string) {
	if err := s.bot.AnswerCallbackQuery(ctx, q.ID, text, false); err != nil {
		s.logger.Warn("telegram answer callback failed", "error", err)
	}
}

func (s *Service) alert(ctx context.Context, q telegram.CallbackQuery, text string) {
	if err := s.bot.AnswerCallbackQuery(ctx, q.ID, text, true); err != nil {
		s.logger.Warn("telegram answer callback failed", "error", err)
	}
}

func parse(data string) (string, string, bool) {
	for _, action := range []string{actionConfirm, actionKeep, actionCancel} {
		if id, ok := strings.CutPrefix(data, action); ok {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return "", "", false
			}
			return action, parsed.String(), true
		}
	}
	return "", "", false
}
