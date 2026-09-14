package postgres_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestTelegramLoginRepositoryLifecycle(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)
	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	repo := postgres.NewTelegramLoginRepository(db)
	now := time.Now().UTC()
	if err := repo.CreateLoginRequest(ctx, "start-1", "poll-1", "K7M3", now, now.Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateLoginRequest() error = %v", err)
	}

	// До привязки чата подтверждать нечего.
	if ok, err := repo.ConfirmLatestForChat(ctx, 777, "+79991234567", "Анна", now); err != nil || ok {
		t.Fatalf("confirm without chat = %v, %v; want false", ok, err)
	}

	request, ok, err := repo.AttachChat(ctx, "start-1", 777, now)
	if err != nil || !ok || request.ConfirmCode != "K7M3" {
		t.Fatalf("AttachChat() = %+v, %v, %v", request, ok, err)
	}
	if _, ok, _ := repo.AttachChat(ctx, "start-1", 888, now); ok {
		t.Fatal("another chat must not take over the request")
	}
	if _, ok, _ := repo.AttachChat(ctx, "start-1", 777, now); !ok {
		t.Fatal("the same chat may open the link again")
	}

	if _, ok, _ := repo.ConsumeConfirmed(ctx, "poll-1", now); ok {
		t.Fatal("pending request must not be consumable")
	}
	if ok, err := repo.ConfirmLatestForChat(ctx, 777, "+79991234567", "Анна", now); err != nil || !ok {
		t.Fatalf("ConfirmLatestForChat() = %v, %v", ok, err)
	}

	// Параллельные опросы: сессию получает ровно один.
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, ok, err := repo.ConsumeConfirmed(ctx, "poll-1", now)
			if err != nil {
				t.Errorf("ConsumeConfirmed() error = %v", err)
				return
			}
			if ok {
				mu.Lock()
				winners++
				mu.Unlock()
				if got.Phone != "+79991234567" || got.FirstName != "Анна" {
					t.Errorf("consumed = %+v", got)
				}
			}
		}()
	}
	wg.Wait()
	if winners != 1 {
		t.Fatalf("winners = %d, want exactly 1", winners)
	}

	status, ok, err := repo.RequestByPollHash(ctx, "poll-1")
	if err != nil || !ok || status.Status != "consumed" {
		t.Fatalf("RequestByPollHash() = %+v, %v, %v", status, ok, err)
	}

	// Истёкший запрос не привязывается и через сутки удаляется.
	if err := repo.CreateLoginRequest(ctx, "start-2", "poll-2", "AAAA", now.Add(-26*time.Hour), now.Add(-25*time.Hour)); err != nil {
		t.Fatalf("create expired: %v", err)
	}
	if _, ok, _ := repo.AttachChat(ctx, "start-2", 777, now); ok {
		t.Fatal("expired request must not attach")
	}
	removed, err := postgres.DeleteStaleTelegramLogins(ctx, db, now)
	if err != nil || removed != 1 {
		t.Fatalf("DeleteStaleTelegramLogins() = %d, %v; want 1", removed, err)
	}
}
