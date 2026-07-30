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
	_ = time.Now() // want "use your injected TimeService field's .Now\\(\\) instead of time.Now\\(\\)"
}

func (s *Service) GoodNow(ctx context.Context) {
	_ = s.clock.Now()
}

func (s *Service) BadSince(ctx context.Context) {
	var start time.Time
	_ = time.Since(start) // want "use your injected TimeService field's .Now\\(\\) instead of time.Since\\(\\)"
}

func (s *Service) GoodSince(ctx context.Context) {
	var start time.Time
	_ = s.clock.Now().(time.Time).Sub(start) // correct: use injected clock
}

func (s *Service) BadUntil(ctx context.Context) {
	var deadline time.Time
	_ = time.Until(deadline) // want "use your injected TimeService field's .Now\\(\\) instead of time.Until\\(\\)"
}

func (s *Service) GoodUntil(ctx context.Context) {
	var deadline time.Time
	_ = deadline.Sub(s.clock.Now().(time.Time)) // correct: use injected clock
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
