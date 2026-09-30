package spanerrors

import (
	"context"

	"github.com/moovfinancial/go-libs/observability/telemetry"
)

type Service struct{}

func (s *Service) BadUnrecorded(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "bad-unrecorded")
	defer span.End()
	if err != nil {
		return err // want "BadUnrecorded creates a span but this return does not record the error"
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

func (s *Service) BadOneBranch(ctx context.Context, a, b error) error {
	_, span := telemetry.StartSpan(ctx, "bad-one-branch")
	defer span.End()
	if a != nil {
		return telemetry.RecordError(ctx, a)
	}
	if b != nil {
		return b // want "BadOneBranch creates a span but this return does not record the error"
	}
	return nil
}

func (s *Service) BadSiblingRecord(ctx context.Context, err error, ok bool) error {
	_, span := telemetry.StartSpan(ctx, "bad-sibling")
	defer span.End()
	if ok {
		err = telemetry.RecordError(ctx, err)
	} else {
		return err // want "BadSiblingRecord creates a span but this return does not record the error"
	}
	return nil
}

func (s *Service) GoodRecordedEarlier(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "good-earlier")
	defer span.End()
	if err != nil {
		err = telemetry.RecordError(ctx, err)
		if ctx != nil {
			return err
		}
		return err
	}
	return nil
}

func (s *Service) GoodDeferred(ctx context.Context, in error) (err error) {
	_, span := telemetry.StartSpan(ctx, "good-deferred")
	defer span.End()
	defer func() {
		if err != nil {
			telemetry.RecordError(ctx, err)
		}
	}()
	return in
}

func (s *Service) OKBeforeSpan(ctx context.Context, err error) error {
	if err != nil {
		return err
	}
	_, span := telemetry.StartSpan(ctx, "ok-before-span")
	defer span.End()
	return nil
}

func (s *Service) OKClosureReturn(ctx context.Context) error {
	_, span := telemetry.StartSpan(ctx, "ok-closure")
	defer span.End()
	f := func(err error) error { return err }
	_ = f
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
