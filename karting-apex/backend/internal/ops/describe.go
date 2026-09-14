package ops

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// DescribeError — описание ошибки для алерта без данных из неё. Сырой текст ошибки может
// содержать значения (номер телефона, id, фрагмент SQL-параметра): он остаётся только в
// журнале сервиса, а в Telegram уходит класс ошибки и место, где она возникла.
//
// Место — первая часть текста до «: ». В коде ошибки оборачиваются как
// fmt.Errorf("claim confirm notifications: %w", err), и эта часть — статичный текст.
func DescribeError(err error) string {
	if err == nil {
		return "неизвестная ошибка"
	}
	kind := errorKind(err)
	if where := staticPrefix(err.Error()); where != "" {
		return kind + " (" + where + ")"
	}
	return kind
}

func errorKind(err error) string {
	var pgErr *pgconn.PgError
	var connectErr *pgconn.ConnectError
	var netErr net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "таймаут"
	case errors.Is(err, context.Canceled):
		return "операция прервана"
	case errors.As(err, &connectErr):
		return "нет соединения с базой"
	case errors.As(err, &pgErr):
		// Код SQLSTATE не содержит данных, в отличие от текста сообщения PostgreSQL.
		return "ошибка PostgreSQL " + pgErr.Code
	case errors.As(err, &netErr):
		return "сетевая ошибка"
	default:
		return "ошибка"
	}
}

func staticPrefix(text string) string {
	prefix, _, found := strings.Cut(text, ": ")
	if !found {
		return ""
	}
	// Префикс из нашего кода короткий и без кавычек; всё иное — уже данные.
	if len(prefix) > 80 || strings.ContainsAny(prefix, `"'`) {
		return ""
	}
	return prefix
}

// DescribePanic — тип значения паники без самого значения: в панику может попасть что угодно.
func DescribePanic(recovered any) string {
	if err, ok := recovered.(error); ok {
		return DescribeError(err)
	}
	return fmt.Sprintf("значение типа %T", recovered)
}
