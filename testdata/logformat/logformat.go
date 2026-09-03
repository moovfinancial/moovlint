package logformat

import (
	"context"
	"fmt"

	"github.com/moovfinancial/go-libs/observability/log"
)

type Service struct {
	logger log.Logger
}

func (s *Service) badLogf(ctx context.Context, err error) {
	s.logger.Logf("save failed: %w", err) // want "%w is only valid in fmt.Errorf"
}

func (s *Service) badLogErrorf(ctx context.Context, err error) {
	s.logger.LogErrorf("save failed: %w", err) // want "%w is only valid in fmt.Errorf"
}

func (s *Service) badChained(ctx context.Context, err error) {
	s.logger.Error().Logf("save failed: %w", err) // want "%w is only valid in fmt.Errorf"
}

func (s *Service) goodFormats(ctx context.Context, err error) {
	s.logger.Logf("save failed: %v", err)
	s.logger.LogErrorf("save failed: %s", err)
	s.logger.Error().Logf("save failed: %v", err)
}

func okErrorf(err error) error {
	return fmt.Errorf("save failed: %w", err)
}
