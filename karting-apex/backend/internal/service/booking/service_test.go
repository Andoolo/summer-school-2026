package booking

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateRejectsInvalidCountsBeforeRepositoryLookup(t *testing.T) {
	repo := &fakeRepo{clientFound: true}
	service := NewService(repo)

	_, err := service.Create(context.Background(), CreateCommand{Token: "token", SlotID: "slot", SeatsCount: 4, RentalCount: 0})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidRequest)
	}
	if repo.clientLookups != 0 {
		t.Fatalf("client lookups = %d, want 0", repo.clientLookups)
	}
}

func TestCreateRejectsUnauthorizedToken(t *testing.T) {
	service := NewService(&fakeRepo{clientFound: false})

	_, err := service.Create(context.Background(), CreateCommand{Token: "token", SlotID: "slot", SeatsCount: 1, RentalCount: 0})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Create() error = %v, want %v", err, ErrUnauthorized)
	}
}

func TestListRejectsInvalidPagination(t *testing.T) {
	service := NewService(&fakeRepo{clientFound: true})

	_, err := service.List(context.Background(), ListCommand{Token: "token", Limit: 101, Offset: 0})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("List() error = %v, want %v", err, ErrInvalidRequest)
	}
}

func TestGetDelegatesForbiddenFromRepository(t *testing.T) {
	service := NewService(&fakeRepo{clientFound: true, getErr: ErrForbidden})

	_, err := service.Get(context.Background(), "token", "booking-id")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Get() error = %v, want %v", err, ErrForbidden)
	}
}

func TestOnChangeFiresOnlyAfterSuccessfulCreateAndCancel(t *testing.T) {
	repo := &fakeRepo{clientFound: true}
	changes := 0
	service := NewService(repo).WithOnChange(func() { changes++ })
	ctx := context.Background()

	if _, err := service.Create(ctx, CreateCommand{Token: "token", SlotID: "slot", SeatsCount: 1}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.Cancel(ctx, "token", "booking"); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if changes != 2 {
		t.Fatalf("changes = %d, want 2", changes)
	}

	repo.createErr = ErrSlotFull
	repo.cancelErr = ErrAlreadyCancelled
	_, _ = service.Create(ctx, CreateCommand{Token: "token", SlotID: "slot", SeatsCount: 1})
	_, _ = service.Cancel(ctx, "token", "booking")
	_, _ = service.Create(ctx, CreateCommand{Token: "token", SlotID: "slot", SeatsCount: 9})
	if changes != 2 {
		t.Fatalf("changes after failures = %d, want still 2", changes)
	}
}

type fakeRepo struct {
	clientFound   bool
	clientLookups int
	getErr        error
	createErr     error
	cancelErr     error
}

func (r *fakeRepo) ClientBySessionTokenHash(context.Context, string) (Client, bool, error) {
	r.clientLookups++
	return Client{ID: "client-id"}, r.clientFound, nil
}

func (r *fakeRepo) Create(context.Context, string, CreateCommand, string, time.Time) (Booking, error) {
	return Booking{}, r.createErr
}

func (r *fakeRepo) List(context.Context, string, ListCommand) (BookingList, error) {
	return BookingList{}, nil
}

func (r *fakeRepo) Get(context.Context, string, string) (Booking, error) {
	return Booking{}, r.getErr
}

func (r *fakeRepo) Cancel(context.Context, string, string, time.Time) (Booking, error) {
	return Booking{}, r.cancelErr
}

func (r *fakeRepo) ActiveBookingsCount(context.Context, string) (int, error) {
	return 0, nil
}
