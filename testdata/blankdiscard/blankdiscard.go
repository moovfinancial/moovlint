package blankdiscard

import (
	"context"
	"io"
)

type Service struct{}

// =============================================================================
// Case 1: Single error return - SHOULD FLAG
// =============================================================================

func singleError() error {
	return nil
}

func (s *Service) BadSingleError(ctx context.Context) {
	_ = singleError() // want "discarded error from singleError"
}

// =============================================================================
// Case 2: Multi-return where one value is error - SHOULD FLAG
// =============================================================================

func multiReturn() (string, error) {
	return "", nil
}

func (s *Service) BadMultiDiscard(ctx context.Context) {
	_, _ = multiReturn() // want "discarded error from multiReturn"
}

// =============================================================================
// Case 3: Close without comment - SHOULD FLAG
// =============================================================================

type closeable struct{}

func (c *closeable) Close() error {
	return nil
}

func (s *Service) BadCloseNoComment(ctx context.Context) {
	conn := &closeable{}
	_ = conn.Close() // want "discarded error from Close"
}

// =============================================================================
// Case 4: Close WITH comment on same line - SHOULD NOT FLAG
// =============================================================================

func (s *Service) GoodCloseSameLine(ctx context.Context) {
	conn := &closeable{}
	_ = conn.Close() // intentionally ignoring close error on read-only resource
}

// =============================================================================
// Case 5: Close WITH comment on preceding line - SHOULD NOT FLAG
// =============================================================================

func (s *Service) GoodClosePrecedingLine(ctx context.Context) {
	conn := &closeable{}
	// We ignore close errors here because the connection is read-only
	_ = conn.Close()
}

// =============================================================================
// Case 6: Non-error type discarded - SHOULD NOT FLAG
// =============================================================================

func nonError() int {
	return 42
}

func (s *Service) GoodNonError(ctx context.Context) {
	_ = nonError()
}

// =============================================================================
// Case 7: Error assigned to named variable - SHOULD NOT FLAG
// =============================================================================

func (s *Service) GoodHandledError(ctx context.Context) {
	err := singleError()
	if err != nil {
		return
	}
}

func (s *Service) GoodHandledMulti(ctx context.Context) {
	_, err := multiReturn()
	if err != nil {
		return
	}
}

// =============================================================================
// Case 8: Interface method returning error - SHOULD FLAG
// =============================================================================

func (s *Service) BadReadCloser(ctx context.Context, rc io.ReadCloser) {
	_ = rc.Close() // want "discarded error from Close"
}

// =============================================================================
// Case 9: Interface method with comment - SHOULD NOT FLAG
// =============================================================================

func (s *Service) GoodReadCloserWithComment(ctx context.Context, rc io.ReadCloser) {
	// Response body already drained; ignore close error
	_ = rc.Close()
}

// =============================================================================
// Case 10: Multi-return with first value used, second (error) discarded - SHOULD FLAG
// =============================================================================

func (s *Service) BadPartialDiscard(ctx context.Context) {
	x, _ := multiReturn() // want "discarded error from multiReturn"
	_ = x
}

// =============================================================================
// Case 11: Non-error discard in multi-return - SHOULD NOT FLAG
// =============================================================================

func multiReturnNoError() (string, int) {
	return "", 0
}

func (s *Service) GoodMultiNonError(ctx context.Context) {
	_, _ = multiReturnNoError()
}

// =============================================================================
// Case 12: Error from method call on interface - SHOULD FLAG
// =============================================================================

type errorReturner interface {
	DoWork() error
}

func (s *Service) BadInterfaceError(ctx context.Context, e errorReturner) {
	_ = e.DoWork() // want "discarded error from DoWork"
}
