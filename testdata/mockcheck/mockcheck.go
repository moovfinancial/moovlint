package mockcheck

import "context"

type PayoutService interface {
	GetPayout(ctx context.Context, id string) (string, error)
	CreatePayout(ctx context.Context, amount int64) (string, error)
}

type ExternalService interface {
	Validate(ctx context.Context, data string) bool
}
