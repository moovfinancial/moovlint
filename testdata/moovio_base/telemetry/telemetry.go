package telemetry

import "context"

func StartSpan(ctx context.Context, spanName string, opts ...any) (context.Context, any) {
	return ctx, nil
}

func StartLinkedRootSpan(ctx context.Context, name string, opts ...any) any {
	return nil
}

func AddEvent(ctx context.Context, name string, opts ...any) {}

func RecordError(ctx context.Context, err error, opts ...any) error { return err }

func RecordErrorAtLow(ctx context.Context, err error, opts ...any) error { return err }

func SetAttributes(ctx context.Context, kv ...any) {}
