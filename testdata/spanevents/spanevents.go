package spanevents

import (
	"github.com/moov-io/base/log"
)

type service struct {
	logger log.Logger
}

func (s *service) DoSomething() {
	s.logger.Info().Log("doing something")     // want "use telemetry.AddEvent or telemetry.RecordError instead of logger.Info\\(\\)\\.Log"
}

func (s *service) DoSomethingElse() {
	s.logger.Warn().Logf("warning: %s", "oh no") // want "use telemetry.AddEvent or telemetry.RecordError instead of logger.Warn\\(\\)\\.Log"
}
