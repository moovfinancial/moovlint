package ctornilguard

import (
	"context"
	"errors"
)

type Store struct {
	clock Clock
}

type Clock interface{ Now() int64 }

func NewStore(clock Clock) (*Store, error) { // want "NewStore stores dependencies without a nil guard: clock"
	return &Store{clock: clock}, nil
}

func NewStoreGuarded(clock Clock) (*Store, error) {
	if clock == nil {
		return nil, errors.New("clock is required")
	}
	return &Store{clock: clock}, nil
}

func NewStoreDefaulted(ctx context.Context, clock Clock) *Store {
	if clock == nil {
		clock = systemClock{}
	}
	_ = ctx
	return &Store{clock: clock}
}

func NewStoreDelegating(clock Clock) (*Store, error) {
	return NewStoreGuarded(clock)
}

func NewStoreNoDeps() *Store {
	return &Store{}
}

type systemClock struct{}

func (systemClock) Now() int64 { return 0 }
