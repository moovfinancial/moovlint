package trace

import "context"

type Span interface {
	End(opts ...any)
	SetName(name string)
	RecordError(err error, opts ...any)
	SetAttributes(kv ...any)
}

func SpanFromContext(ctx context.Context) Span {
	return nil
}

func StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, nil
}
