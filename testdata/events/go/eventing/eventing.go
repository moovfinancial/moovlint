package eventing

import (
	"context"

	v1 "github.com/moovfinancial/events/go/events/v1"
)

type EventHandlerContext func(ctx context.Context, event *v1.Event) error

type Record struct{}

type RecordHandler func(ctx context.Context, r []*Record) error

type EventMessage struct {
	Event *v1.Event
}

type EventMessageHandler func(batchCtx context.Context, events []*EventMessage) error

type RawMessage struct{}

type RawMessageHandler func(batchCtx context.Context, msgs []*RawMessage) error

type EventHeadersMessage struct{}

type EventHeaderMessageHandler func(batchCtx context.Context, events []*EventHeadersMessage) error

type InitializedEvent struct{}

type RawBytes struct{}

// Producer mirrors eventing.Producer. Produce/ProduceBytes open a "producing"
// span internally, so callers must not wrap them in a manual child span.
type Producer interface {
	Event(ctx context.Context, topic string, idOrKey string, data *v1.EventData) (InitializedEvent, error)
	Produce(ctx context.Context, datas ...InitializedEvent) ([]InitializedEvent, error)
	ProduceBytes(ctx context.Context, msgs ...RawBytes) error
	Flush(ctx context.Context) error
}

type Writer interface {
	Insert(ctx context.Context, name string) error
}

func AddEventContextHandler(handlers ...EventHandlerContext) {}
