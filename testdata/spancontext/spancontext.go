package spancontext

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type Service struct{}

func (s *Service) BadDirectEnd(ctx context.Context) {
	trace.SpanFromContext(ctx).End() // want "do not call End on a span retrieved from context"
}

func (s *Service) BadDirectSetName(ctx context.Context) {
	trace.SpanFromContext(ctx).SetName("foo") // want "do not call SetName on a span retrieved from context"
}

func (s *Service) BadVarEnd(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	span.End() // want "do not call End on 'span' which is a span retrieved from context"
}

func (s *Service) OK(ctx context.Context) {
	trace.SpanFromContext(ctx).RecordError(nil)
}

// OwnedEnd uses the same local name as BadVarEnd but its span comes from
// StartSpan, not from context. A sibling function's context-derived span of
// the same name must not flag it.
func (s *Service) OwnedEnd(ctx context.Context) {
	_, span := trace.StartSpan(ctx, "owned")
	defer span.End()
}
