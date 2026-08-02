package booking

import (
	"testing"
	"time"
)

func TestCancellationStatusBoundaries(t *testing.T) {
	start := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		now    time.Time
		status string
		ok     bool
	}{
		{name: "two hours plus one second", now: start.Add(-2*time.Hour - time.Second), status: "cancelled", ok: true},
		{name: "exactly two hours", now: start.Add(-2 * time.Hour), status: "cancelled", ok: true},
		{name: "one second under two hours", now: start.Add(-2*time.Hour + time.Second), status: "late_cancel", ok: true},
		{name: "after start", now: start.Add(time.Second), status: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, ok := CancellationStatus(tt.now, start)
			if status != tt.status || ok != tt.ok {
				t.Fatalf("CancellationStatus() = %q, %v; want %q, %v", status, ok, tt.status, tt.ok)
			}
		})
	}
}
