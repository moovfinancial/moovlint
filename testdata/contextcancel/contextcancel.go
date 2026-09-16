package contextcancel

import (
	"context"
	"time"
)

type Service struct{}

func (s *Service) BadCancel(ctx context.Context) { // want "context cancel function 'cancel' is never deferred"
	ctx, cancel := context.WithCancel(ctx)
	_ = ctx
	_ = cancel
}

func (s *Service) GoodCancel(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	_ = ctx
}

func (s *Service) BadTimeout(ctx context.Context) { // want "context cancel function 'cancel' is never deferred"
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_ = ctx
	_ = cancel
}

func (s *Service) GoodTimeout(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = ctx
}

// ReturnedCancel hands the cancel function to its caller; the caller owns it.
func (s *Service) ReturnedCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	return ctx, cancel
}

// ClosureCancel stops the context from a returned shutdown closure.
func (s *Service) ClosureCancel() func() {
	ctx, stop := context.WithCancel(context.Background())
	_ = ctx
	return func() {
		stop()
	}
}
