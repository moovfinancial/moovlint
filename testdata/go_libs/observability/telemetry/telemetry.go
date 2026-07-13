package telemetry

import "context"

type Span struct{}

func (s Span) End(opts ...any)                 {}
func (s Span) SetName(name string)             {}
func (s Span) RecordError(err error, opts ...any) {}

func StartSpan(ctx context.Context, spanName string, opts ...any) (context.Context, Span) {
	return ctx, Span{}
}

func StartLinkedRootSpan(ctx context.Context, name string, opts ...any) Span {
	return Span{}
}

func AddEvent(ctx context.Context, name string, opts ...any) {}

func RecordError(ctx context.Context, err error, opts ...any) error { return err }

func RecordErrorAtLow(ctx context.Context, err error, opts ...any) error { return err }

func SetAttributes(ctx context.Context, kv ...any) {}
