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
