package spanerrors

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/moovfinancial/errors"
	"github.com/moovfinancial/go-libs/observability/telemetry"
)

type Service struct{}

func (s *Service) BadOrigin(ctx context.Context, ok bool) error {
	_, span := telemetry.StartSpan(ctx, "bad-origin")
	defer span.End()
	if !ok {
		return stderrors.New("not ok") // want "BadOrigin creates this error but does not record it"
	}
	return nil
}

func (s *Service) BadErrorfNoWrap(ctx context.Context, id string) error {
	_, span := telemetry.StartSpan(ctx, "bad-errorf")
	defer span.End()
	return fmt.Errorf("payout %s missing", id) // want "BadErrorfNoWrap creates this error but does not record it"
}

func (s *Service) BadFlagged(ctx context.Context) (int, error) {
	_, span := telemetry.StartSpan(ctx, "bad-flagged")
	defer span.End()
	return 0, errors.Flag(stderrors.New("bad state"), errors.NotValidState) // want "BadFlagged creates this error but does not record it"
}

func (s *Service) OKCalleeError(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "ok-callee")
	defer span.End()
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) OKWrappedCallee(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "ok-wrapped")
	defer span.End()
	return fmt.Errorf("loading: %w", err)
}

func (s *Service) OKFlaggedCallee(ctx context.Context, err error) error {
	_, span := telemetry.StartSpan(ctx, "ok-flagged-callee")
	defer span.End()
	return errors.Flag(err, errors.NotFound)
}

func (s *Service) OKHelper(ctx context.Context) error {
	_, span := telemetry.StartSpan(ctx, "ok-helper")
	defer span.End()
	return notFound(ctx, "missing")
}

func (s *Service) GoodRecorded(ctx context.Context) error {
	_, span := telemetry.StartSpan(ctx, "good-recorded")
	defer span.End()
	return telemetry.RecordError(ctx, stderrors.New("x"))
}

func (s *Service) GoodAtLow(ctx context.Context) error {
	_, span := telemetry.StartSpan(ctx, "good-at-low")
	defer span.End()
	return telemetry.RecordErrorAtLow(ctx, errors.Flag(stderrors.New("x"), errors.NotFound))
}

func (s *Service) GoodSpanMethod(ctx context.Context) error {
	_, span := telemetry.StartSpan(ctx, "good-span-method")
	defer span.End()
	err := stderrors.New("x")
	span.RecordError(err)
	return err
}

func (s *Service) BadSiblingRecord(ctx context.Context, ok bool) error {
	_, span := telemetry.StartSpan(ctx, "bad-sibling")
	defer span.End()
	if ok {
		telemetry.RecordError(ctx, stderrors.New("a"))
	} else {
		return stderrors.New("b") // want "BadSiblingRecord creates this error but does not record it"
	}
	return nil
}

func (s *Service) GoodRecordedEarlier(ctx context.Context, ok bool) error {
	_, span := telemetry.StartSpan(ctx, "good-earlier")
	defer span.End()
	if !ok {
		telemetry.RecordError(ctx, stderrors.New("a"))
		return stderrors.New("a")
	}
	return nil
}

func (s *Service) GoodDeferred(ctx context.Context) (err error) {
	_, span := telemetry.StartSpan(ctx, "good-deferred")
	defer span.End()
	defer func() {
		if err != nil {
			telemetry.RecordError(ctx, err)
		}
	}()
	return stderrors.New("x")
}

func (s *Service) OKBeforeSpan(ctx context.Context, ok bool) error {
	if !ok {
		return stderrors.New("x")
	}
	_, span := telemetry.StartSpan(ctx, "ok-before-span")
	defer span.End()
	return nil
}

func (s *Service) OKClosureReturn(ctx context.Context) error {
	_, span := telemetry.StartSpan(ctx, "ok-closure")
	defer span.End()
	f := func() error { return stderrors.New("x") }
	_ = f
	return nil
}

func (s *Service) OKNoSpan(ctx context.Context) error {
	return stderrors.New("x")
}

func notFound(ctx context.Context, msg string) error {
	return telemetry.RecordErrorAtLow(ctx, errors.Flag(stderrors.New(msg), errors.NotFound))
}
