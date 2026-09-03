package spanerrors

import (
	"context"

	"github.com/moovfinancial/go-libs/observability/telemetry"
)

type Service struct{}

func (s *Service) BadUnrecorded(ctx context.Context, err error) error { // want "BadUnrecorded creates a span but returns errors without recording"
	_, span := telemetry.StartSpan(ctx, "bad-unrecorded")
	defer span.End()
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GoodRecorded(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "good-recorded")
	defer span.End()
	if err != nil {
		return telemetry.RecordError(ctx, err)
	}
	return nil
}

func (s *Service) GoodAtLow(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "good-at-low")
	defer span.End()
	if err != nil {
		return telemetry.RecordErrorAtLow(ctx, err)
	}
	return nil
}

func (s *Service) GoodSpanMethod(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "good-span-method")
	defer span.End()
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (s *Service) OKNoSpan(ctx context.Context, err error) error {
	return err
}

func (s *Service) OKOnlyNilReturns(ctx context.Context) error {
	_, span := telemetry.StartSpan(ctx, "ok-only-nil")
	defer span.End()
	_ = span
	return nil
}
