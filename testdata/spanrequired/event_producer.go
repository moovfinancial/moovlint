package spanrequired

import (
	"context"
	"fmt"

	"github.com/moovfinancial/events/go/eventing"
	v1 "github.com/moovfinancial/events/go/events/v1"
)

// eventProducer only hands work to eventing.Producer, which opens its own
// "producing" span. A manual span here would duplicate it.
type eventProducer struct {
	producer eventing.Producer
	topic    string
}

// Exempt: every context-carrying call is auto-instrumented.
func (e *eventProducer) SandboxIssuingAuthorization(ctx context.Context, id string) error {
	ie, err := e.producer.Event(ctx, e.topic, id, &v1.EventData{Name: "auth"})
	if err != nil {
		return fmt.Errorf("initializing event: %w", err)
	}

	if _, err := e.producer.Produce(ctx, ie); err != nil {
		return fmt.Errorf("producing event: %w", err)
	}

	return nil
}

// Exempt: a non-context helper call does not count as untraced work.
func (e *eventProducer) SandboxIssuingReversal(ctx context.Context, id string) error {
	data := buildPayload(id)
	ie, err := e.producer.Event(ctx, e.topic, id, data)
	if err != nil {
		return err
	}
	_, err = e.producer.Produce(ctx, ie)
	return err
}

func buildPayload(id string) *v1.EventData { return &v1.EventData{Name: id} }
