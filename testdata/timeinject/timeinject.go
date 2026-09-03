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
	_ = time.Now() // want "use injected stime.TimeService instead of time.Now"
}

func (s *Service) GoodNow(ctx context.Context) {
	_ = s.clock.Now()
}

type NoTimeService struct{}

func (s *NoTimeService) OKNow(ctx context.Context) {
	_ = time.Now()
}

func BadNowAsValue(clock func() time.Time) func() time.Time {
	clock = time.Now // want "time.Now is passed as the wall clock"
	return clock
}

func OKNowCalled() time.Time {
	return time.Now()
}
