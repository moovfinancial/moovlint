package spanlifecycle

import (
	"context"

	"github.com/moovfinancial/go-libs/observability/telemetry"
)

type Service struct{}

func (s *Service) MissingEnd(ctx context.Context) { // want "span assigned to 'span' is never ended"
	ctx, span := telemetry.StartSpan(ctx, "missing-end")
	_ = ctx
	_ = span
}

func (s *Service) HasEnd(ctx context.Context) {
	ctx, span := telemetry.StartSpan(ctx, "has-end")
	defer span.End()
	_ = ctx
}

func (s *Service) LinkedRootMissing(ctx context.Context) { // want "span assigned to 'span' is never ended"
	span := telemetry.StartLinkedRootSpan(ctx, "linked-root")
	_ = span
}

func (s *Service) LinkedRootHasEnd(ctx context.Context) {
	span := telemetry.StartLinkedRootSpan(ctx, "linked-root")
	defer span.End()
}
