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
