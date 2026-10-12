package spanrequired

import (
	"context"
	"fmt"
)

type QuoteService struct {
	pricing pricer
}

// Exempt: hands off to a single delegate; the span belongs there.
func (s *QuoteService) GetQuote(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("missing id")
	}
	quote, err := s.pricing.Price(ctx, id)
	if err != nil {
		return "", fmt.Errorf("getting quote: %w", err)
	}
	return quote, nil
}

// Still reported: two units of work are orchestration, not a pass-through.
func (s *QuoteService) CompareQuotes(ctx context.Context, a, b string) (bool, error) { // want "exported method QuoteService\\.CompareQuotes takes context but does not start a telemetry span"
	qa, err := s.pricing.Price(ctx, a)
	if err != nil {
		return false, err
	}
	qb, err := s.pricing.Price(ctx, b)
	if err != nil {
		return false, err
	}
	return qa == qb, nil
}

// Still reported: one call site in a loop runs once per item.
func (s *QuoteService) QuoteAll(ctx context.Context, ids []string) error { // want "exported method QuoteService\\.QuoteAll takes context but does not start a telemetry span"
	for _, id := range ids {
		if _, err := s.pricing.Price(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Still reported: the single direct call reaches a helper doing two units of
// untraced work, so the method is not a pass-through.
func (s *QuoteService) RefreshQuote(ctx context.Context, id string) error { // want "exported method QuoteService\\.RefreshQuote takes context but does not start a telemetry span"
	return s.refresh(ctx, id)
}

func (s *QuoteService) refresh(ctx context.Context, id string) error {
	if _, err := s.pricing.Price(ctx, id); err != nil {
		return err
	}
	_, err := s.pricing.Price(ctx, id+"-fee")
	return err
}
