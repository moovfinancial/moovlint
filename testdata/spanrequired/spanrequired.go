package spanrequired

import (
	"context"
)

// PayoutService has an exported method without a span.
type PayoutService struct{}

func (s *PayoutService) GetPayout(ctx context.Context, id string) (string, error) { // want "exported method PayoutService\\.GetPayout takes context but does not start a telemetry span"
	return id, nil
}
