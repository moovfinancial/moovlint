package spanrequired

import (
	"context"

	"github.com/moovfinancial/events/go/eventing"
	obsql "github.com/moovfinancial/go-libs/observability/sql"
)

type pricingClient struct{}

// pricer is implemented outside this package, so the analyzer cannot follow
// a call to it and counts it as untraced work.
type pricer interface {
	Price(ctx context.Context, id string) (string, error)
}

func (p *pricingClient) Quote(ctx context.Context, id string) (string, error) { // want "exported method pricingClient\\.Quote takes context but does not start a telemetry span"
	return id, nil
}

type TransferService struct {
	producer eventing.Producer
	db       *obsql.DB
	pricing  pricer
}

// Still reported: the pricing call is real work that nothing else traces, so
// this method is a genuine unit of work and needs its own span.
func (s *TransferService) CreateTransfer(ctx context.Context, id string) error { // want "exported method TransferService\\.CreateTransfer takes context but does not start a telemetry span"
	if _, err := s.pricing.Price(ctx, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, "INSERT", id)
	return err
}

// Still reported: untraced work after the instrumented call must not be
// masked by the exemption.
func (s *TransferService) PublishTransfer(ctx context.Context, id string) error { // want "exported method TransferService\\.PublishTransfer takes context but does not start a telemetry span"
	if err := s.producer.Flush(ctx); err != nil {
		return err
	}
	_, err := s.pricing.Price(ctx, id)
	return err
}
