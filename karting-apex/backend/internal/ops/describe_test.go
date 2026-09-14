package ops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestDescribeErrorHidesData(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "22P02", Message: `invalid input syntax for type uuid: "+79991234567"`}
	cases := []struct {
		err       error
		want      string
		forbidden string
	}{
		{fmt.Errorf("claim confirm notifications: %w", pgErr), "ошибка PostgreSQL 22P02 (claim confirm notifications)", "79991234567"},
		{fmt.Errorf("delete stale telegram logins: %w", context.DeadlineExceeded), "таймаут (delete stale telegram logins)", ""},
		{errors.New(`lookup for "+79991234567": boom`), "ошибка", "79991234567"},
		{errors.New("без двоеточия"), "ошибка", "без двоеточия"},
	}
	for _, c := range cases {
		got := DescribeError(c.err)
		if got != c.want {
			t.Errorf("DescribeError(%v) = %q, want %q", c.err, got, c.want)
		}
		if c.forbidden != "" && strings.Contains(got, c.forbidden) {
			t.Errorf("DescribeError leaks %q: %q", c.forbidden, got)
		}
	}
}

func TestDescribePanicHidesValue(t *testing.T) {
	if got := DescribePanic("телефон +79991234567"); strings.Contains(got, "7999") || got != "значение типа string" {
		t.Fatalf("DescribePanic(string) = %q", got)
	}
	if got := DescribePanic(fmt.Errorf("parse booking: %w", errors.New(`bad "x"`))); got != "ошибка (parse booking)" {
		t.Fatalf("DescribePanic(error) = %q", got)
	}
}
