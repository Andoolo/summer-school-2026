package ops

import (
	"context"
	"errors"
	"time"

	"summer-school-2026/backend/internal/domain"
)

type StateStore interface {
	// State возвращает значение ("" — не было).
	State(ctx context.Context, key string) (string, error)
	SetState(ctx context.Context, key, value string) error
}

// AnnounceVersion сообщает администратору о новой версии — один раз на версию. На каждое
// пробуждение Render не сообщает: сервис просыпается по многу раз в день.
//
// Версия помечается объявленной только после доставки: если Telegram недоступен, объявление
// повторится при следующем запуске. Одновременный старт двух экземпляров может объявить
// версию дважды — для сервиса в одном экземпляре это приемлемо.
func AnnounceVersion(ctx context.Context, store StateStore, recorder *Recorder, version string, now time.Time) error {
	if version == "" || recorder == nil {
		return nil
	}
	previous, err := store.State(ctx, "announced_version")
	if err != nil {
		return err
	}
	if previous == version {
		return nil
	}
	text := "🚀 Апекс обновлён: версия " + shortVersion(version) + ", запущен в " + now.In(domain.ClubZone).Format("15:04") + " (мск)."
	if previous != "" {
		text += "\nПредыдущая версия: " + shortVersion(previous) + "."
	}
	if err := recorder.Notify(ctx, text); err != nil {
		if errors.Is(err, ErrNoAdmin) {
			return nil
		}
		return err
	}
	return store.SetState(ctx, "announced_version", version)
}
