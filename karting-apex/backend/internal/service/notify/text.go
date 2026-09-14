package notify

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Трассы в Москве; перехода на летнее время там нет, поэтому хватает фиксированного
// смещения — и образ не зависит от наличия tzdata.
var moscow = time.FixedZone("MSK", 3*60*60)

var (
	weekdays = [...]string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}
	months   = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
)

const footer = "Отключить уведомления: /stop"

// Text собирает сообщение без разметки: названия и точки сбора вставляются как есть.
func Text(n Notice, now time.Time) string {
	var b strings.Builder
	switch n.Kind {
	case KindConfirm:
		b.WriteString("✅ Бронь подтверждена\n\n")
		writeRace(&b, n)
		b.WriteString("🎟 " + seatsLine(n.SeatsCount, n.RentalCount) + "\n")
		if n.PriceTotal > 0 {
			b.WriteString("💳 Сумма: " + rubles(n.PriceTotal) + "\n")
		}
		if ReminderExpected(n.CreatedAt, n.StartAt) {
			b.WriteString("\nНапомню о заезде за 2 часа до старта.\n")
		}
	case KindReminder:
		b.WriteString("⏰ Заезд " + untilStart(n.StartAt.Sub(now)) + "\n\n")
		writeRace(&b, n)
		b.WriteString("🎟 " + seatsLine(n.SeatsCount, n.RentalCount) + "\n")
	case KindCancel:
		b.WriteString("❌ Бронь отменена\n\n")
		writeRace(&b, n)
		if n.Status == "late_cancel" {
			b.WriteString("\nПоздняя отмена — меньше чем за 2 часа до старта: место в заезде не освободилось. Штраф не взимается.\n")
		}
	}
	b.WriteString("\n" + footer)
	return b.String()
}

// ReminderExpected — получит ли бронь напоминание (то же правило, что в выборке).
func ReminderExpected(createdAt, startAt time.Time) bool {
	return !createdAt.After(startAt.Add(-ReminderMinAdvance))
}

func writeRace(b *strings.Builder, n Notice) {
	b.WriteString("🏁 " + n.RouteName + "\n")
	b.WriteString("🗓 " + FormatStart(n.StartAt) + "\n")
	if n.MeetingPoint != "" {
		b.WriteString("📍 Сбор: " + n.MeetingPoint + "\n")
	}
	if n.InstructorName != "" {
		b.WriteString("🚩 Маршал: " + n.InstructorName + "\n")
	}
}

// FormatStart — «суббота, 20 сентября, 18:30 (мск)».
func FormatStart(t time.Time) string {
	local := t.In(moscow)
	return fmt.Sprintf("%s, %d %s, %02d:%02d (мск)", weekdays[local.Weekday()], local.Day(), months[local.Month()-1], local.Hour(), local.Minute())
}

func seatsLine(seats, rental int) string {
	line := "Мест: " + strconv.Itoa(seats)
	if rental > 0 {
		line += ", с экипировкой: " + strconv.Itoa(rental)
	}
	return line
}

// untilStart — «через 2 ч», «через 1 ч 15 мин», «через 40 мин», «вот-вот начнётся».
func untilStart(left time.Duration) string {
	if left < time.Minute {
		return "вот-вот начнётся"
	}
	minutes := int(left.Round(time.Minute) / time.Minute)
	switch {
	case minutes < 60:
		return fmt.Sprintf("через %d мин", minutes)
	case minutes%60 == 0:
		return fmt.Sprintf("через %d ч", minutes/60)
	default:
		return fmt.Sprintf("через %d ч %d мин", minutes/60, minutes%60)
	}
}

// rubles — «5 800 ₽» с неразрывным пробелом между разрядами.
func rubles(amount int) string {
	digits := strconv.Itoa(amount)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteRune('\u00a0')
		}
		b.WriteRune(r)
	}
	return b.String() + " ₽"
}
