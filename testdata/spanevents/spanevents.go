package spanevents

import (
	"context"

	"github.com/moov-io/base/log"
)

type service struct {
	logger log.Logger
}

func (s *service) DoSomething(ctx context.Context) {
	s.logger.Info().Log("doing something") // want "use telemetry.AddEvent or telemetry.RecordError instead of logger.Info\\(\\)\\.Log"
}

func (s *service) DoSomethingElse(ctx context.Context) {
	s.logger.Warn().Logf("warning: %s", "oh no") // want "use telemetry.AddEvent or telemetry.RecordError instead of logger.Warn\\(\\)\\.Log"
}

// BootWithoutContext has no context parameter, so the telemetry suggestion
// cannot be applied. Lifecycle logs stay logs.
func (s *service) BootWithoutContext() {
	s.logger.Info().Log("booting")
	s.logger.Info().Logf("listening on %s", ":8080")
}

// ClosureWithContext takes the context through a closure parameter.
func (s *service) ClosureWithContext() func(context.Context) {
	return func(ctx context.Context) {
		s.logger.Info().Log("in closure") // want "use telemetry.AddEvent or telemetry.RecordError instead of logger.Info\\(\\)\\.Log"
	}
}

// ClosureWithoutContext inherits no context from its enclosing function.
func (s *service) ClosureWithoutContext() func() {
	return func() {
		s.logger.Info().Log("in closure")
	}
}
