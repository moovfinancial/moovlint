package spanrequired

import (
	"context"

	"github.com/moovfinancial/go-libs/observability/telemetry"
)

type LedgerService struct {
	pricing *pricingClient
}

// Exempt: enriches an existing span via the context helper, the engineering-guide
// telemetry doc's recommended alternative to a child span at a non-entry point.
func (s *LedgerService) PostEntry(ctx context.Context, id string) error {
	telemetry.SetAttributes(ctx, "entry_id", id)
	_, err := s.pricing.Quote(ctx, id)
	return err
}

// Exempt: retrieves the existing span and adds an event to it.
func (s *LedgerService) ReverseEntry(ctx context.Context, id string) error {
	span := telemetry.SpanFromContext(ctx)
	span.AddEvent("reverseEntry")
	_, err := s.pricing.Quote(ctx, id)
	return err
}

// Still reported: recording an error is not span instrumentation, it happens
// on nearly every error path.
func (s *LedgerService) VoidEntry(ctx context.Context, id string) error { // want "exported method LedgerService\\.VoidEntry takes context but does not start a telemetry span"
	if _, err := s.pricing.Quote(ctx, id); err != nil {
		return telemetry.RecordError(ctx, err)
	}
	return nil
}
