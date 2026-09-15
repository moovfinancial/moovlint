package spanname

import (
	"context"

	"github.com/moovfinancial/go-libs/observability/telemetry"
)

func badStartSpan(ctx context.Context) {
	_, span := telemetry.StartSpan(ctx, "cron.Handler") // want `span name "cron.Handler" must be lower-kebab-case`
	defer span.End()
}

func badSetName(ctx context.Context) {
	_, span := telemetry.StartSpan(ctx, "cron-handler")
	defer span.End()
	span.SetName("validate merchant category") // want `span name "validate merchant category" must be lower-kebab-case`
}

func badLinkedRoot(ctx context.Context) {
	span := telemetry.StartLinkedRootSpan(ctx, "Transfer Started") // want `span name "Transfer Started" must be lower-kebab-case`
	defer span.End()
}

func goodNames(ctx context.Context) {
	_, span := telemetry.StartSpan(ctx, "cron-handler")
	defer span.End()
	span.SetName("validate-merchant-category-restrictions")
	telemetry.StartLinkedRootSpan(ctx, "transfer-started")
}

func goodNonLiteral(ctx context.Context, name string) {
	_, span := telemetry.StartSpan(ctx, name)
	defer span.End()
}
