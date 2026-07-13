package mid

import "context"

type Account struct{}
type Payout struct{}
type Token struct{}
type User struct{}
type Session struct{}
type Request struct{}

type ID[T any] struct{ raw string }

func (id ID[T]) String() string { return id.raw }

func ParseID[T any](s string) (ID[T], error) {
	return ID[T]{raw: s}, nil
}

func MustParseID[T any](s string) ID[T] {
	return ID[T]{raw: s}
}

func NewRandomID[T any](ctx context.Context) ID[T] {
	return ID[T]{}
}

func IsEmpty[T any](id ID[T]) bool {
	return id.raw == ""
}

func IsID[T any](id ID[T]) bool {
	return id.raw != ""
}

func IsRequiredID[T any](id ID[T]) bool {
	return id.raw != ""
}
