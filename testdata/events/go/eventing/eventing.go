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

type Writer interface {
	Insert(ctx context.Context, name string) error
}

func AddEventContextHandler(handlers ...EventHandlerContext) {}
