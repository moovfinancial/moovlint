package timers

import (
	"context"
	"time"

	stime "github.com/moov-io/base/stime"
)

type Service struct {
	clock stime.TimeService
}

func (s *Service) BadTimers(ctx context.Context, f func()) {
	_ = time.NewTimer(time.Second)     // want `time.NewTimer schedules on the wall clock`
	_ = time.After(time.Second)        // want `time.After schedules on the wall clock`
	_ = time.AfterFunc(time.Second, f) // want `time.AfterFunc schedules on the wall clock`
	_ = time.Tick(time.Second)         // want `time.Tick schedules on the wall clock`
	_ = time.NewTicker(time.Second)    // want `time.NewTicker schedules on the wall clock`
	_ = time.Now()                     // want `instead of time.Now`
}

type NoTimeService struct{}

func (s *NoTimeService) OKTimer(ctx context.Context) {
	_ = time.NewTimer(time.Second)
}
