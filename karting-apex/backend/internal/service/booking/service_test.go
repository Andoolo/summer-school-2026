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

type fakeRepo struct {
	clientFound   bool
	clientLookups int
	getErr        error
}

func (r *fakeRepo) ClientBySessionTokenHash(context.Context, string) (Client, bool, error) {
	r.clientLookups++
	return Client{ID: "client-id"}, r.clientFound, nil
}

func (r *fakeRepo) Create(context.Context, string, CreateCommand, string, time.Time) (Booking, error) {
	return Booking{}, nil
}

func (r *fakeRepo) List(context.Context, string, ListCommand) (BookingList, error) {
	return BookingList{}, nil
}

func (r *fakeRepo) Get(context.Context, string, string) (Booking, error) {
	return Booking{}, r.getErr
}

func (r *fakeRepo) Cancel(context.Context, string, string, time.Time) (Booking, error) {
	return Booking{}, nil
}
