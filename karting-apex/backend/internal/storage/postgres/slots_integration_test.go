package postgres_test

import (
	"context"
	"testing"

	"summer-school-2026/backend/internal/storage/postgres"
	"summer-school-2026/backend/internal/storage/postgres/testutil"
)

func TestSlotRepositoryListReadsSeedSlots(t *testing.T) {
	databaseURL := testutil.PrepareDatabase(t)

	ctx := context.Background()
	db, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(db.Close)

	repo := postgres.NewSlotRepository(db)
	list, err := repo.List(ctx, postgres.SlotFilters{Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list.Items) != 2 {
		t.Fatalf("len(slots) = %d, want %d", len(list.Items), 2)
	}
	if list.Total != 2 {
		t.Fatalf("total = %d, want %d", list.Total, 2)
	}
	if list.Items[0].ID != "55555555-5555-5555-5555-555555555555" {
		t.Fatalf("first slot id = %q", list.Items[0].ID)
	}
	if list.Items[0].FreeSeats != 8 || list.Items[0].FreeRentalBoards != 12 {
		t.Fatalf("unexpected availability: seats=%d boards=%d", list.Items[0].FreeSeats, list.Items[0].FreeRentalBoards)
	}
	if list.Items[0].RouteName == "" || list.Items[0].InstructorName == "" {
		t.Fatal("slot must include route and instructor data")
	}
}
