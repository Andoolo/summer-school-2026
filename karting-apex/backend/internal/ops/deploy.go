package ops

import (
	"context"
	"time"
)

type StateStore interface {
	// SwapState записывает значение и возвращает прежнее ("" — не было).
	SwapState(ctx context.Context, key, value string) (string, error)
}

var moscow = time.FixedZone("MSK", 3*60*60)

// AnnounceVersion сообщает администратору о новой версии — один раз на версию. На каждое
// пробуждение Render не сообщает: сервис просыпается по многу раз в день.
func AnnounceVersion(ctx context.Context, store StateStore, recorder *Recorder, version string, now time.Time) error {
	if version == "" || recorder == nil {
		return nil
	}
	previous, err := store.SwapState(ctx, "announced_version", version)
	if err != nil {
		return err
	}
	if previous == version {
		return nil
	}
	text := "🚀 Апекс обновлён: версия " + shortVersion(version) + ", запущен в " + now.In(moscow).Format("15:04") + " (мск)."
	if previous != "" {
		text += "\nПредыдущая версия: " + shortVersion(previous) + "."
	}
	recorder.Alert("deploy:"+version, text)
	return nil
}
