package timeinject

import (
	"context"
	"time"

	stime "github.com/moov-io/base/stime"
)

type Service struct {
	clock stime.TimeService
}

func (s *Service) BadNow(ctx context.Context) {
	_ = time.Now() // want "use svc.time.Now\\(\\) instead of time.Now\\(\\); TimeService does not expose Now/Since/Until wrappers"
}

func (s *Service) GoodNow(ctx context.Context) {
	_ = s.clock.Now()
}

func (s *Service) BadSince(ctx context.Context) {
	var start time.Time
	_ = time.Since(start) // want "use svc.time.Now\\(\\) instead of time.Since\\(\\); TimeService does not expose Now/Since/Until wrappers"
}

func (s *Service) GoodSince(ctx context.Context) {
	// Manually compute duration using s.clock.Now().Sub(start)
	_ = ctx
}

func (s *Service) BadUntil(ctx context.Context) {
	var deadline time.Time
	_ = time.Until(deadline) // want "use svc.time.Now\\(\\) instead of time.Until\\(\\); TimeService does not expose Now/Since/Until wrappers"
}

func (s *Service) GoodUntil(ctx context.Context) {
	// Manually compute duration using deadline.Sub(s.clock.Now())
	_ = ctx
}

type NoTimeService struct{}

func (s *NoTimeService) OKNow(ctx context.Context) {
	_ = time.Now()
}

func (s *NoTimeService) OKSince(ctx context.Context) {
	_ = time.Since(time.Now())
}

func (s *NoTimeService) OKUntil(ctx context.Context) {
	_ = time.Until(time.Now().Add(time.Hour))
}
