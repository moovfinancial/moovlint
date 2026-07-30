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
	_ = singleError() // want "discarding error from singleError via blank assignment"
}

// =============================================================================
// Case 2: Multi-return where one value is error - SHOULD FLAG
// =============================================================================

func multiReturn() (string, error) {
	return "", nil
}

func (s *Service) BadMultiDiscard(ctx context.Context) {
	_, _ = multiReturn() // want "discarding error from multiReturn via blank assignment"
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
	_ = conn.Close() // want "discarding error from Close via blank assignment"
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
	_ = rc.Close() // want "discarding error from Close via blank assignment"
}

// =============================================================================
// Case 9: Interface method with comment - SHOULD NOT FLAG
// =============================================================================

func (s *Service) GoodReadCloserWithComment(ctx context.Context, rc io.ReadCloser) {
	// Response body already drained; ignore close error
	_ = rc.Close()
}

// =============================================================================
// Case 10: Short declaration (:=) - SHOULD NOT FLAG (only token.ASSIGN is checked)
// =============================================================================

func (s *Service) GoodShortDeclaration(ctx context.Context) {
	x, _ := multiReturn()
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
	_ = e.DoWork() // want "discarding error from DoWork via blank assignment"
}

// =============================================================================
// Case 13: Non-call RHS (variable assignment) - SHOULD NOT FLAG
// =============================================================================

func (s *Service) GoodNonCallRHS(ctx context.Context) {
	var err error
	_ = err // assigning variable, not calling function - no flag
}

// =============================================================================
// Case 14: Error-first multi-return pattern - SHOULD FLAG
// =============================================================================

func errorFirst() (error, string) {
	return nil, ""
}

func (s *Service) BadErrorFirst(ctx context.Context) {
	_, _ = errorFirst() // want "discarding error from errorFirst via blank assignment"
}

// =============================================================================
// Case 15: Generic function returning error - SHOULD FLAG
// =============================================================================

func genericFunc[T any]() error {
	return nil
}

func (s *Service) BadGenericFunc(ctx context.Context) {
	_ = genericFunc[int]() // want "discarding error from genericFunc via blank assignment"
}

// =============================================================================
// Case 16: Generic function with multiple type params returning error - SHOULD FLAG
// =============================================================================

func genericMultiFunc[T, U any]() error {
	return nil
}

func (s *Service) BadGenericMultiFunc(ctx context.Context) {
	_ = genericMultiFunc[int, string]() // want "discarding error from genericMultiFunc via blank assignment"
}

// =============================================================================
// Case 17: Parallel multi-call assignment - SHOULD FLAG
// =============================================================================

func (s *Service) BadParallelMultiCall(ctx context.Context) {
	_, _ = singleError(), singleError() // want "discarding error from singleError via blank assignment" "discarding error from singleError via blank assignment"
}

// =============================================================================
// Case 18: Close with section separator comment - SHOULD FLAG
// =============================================================================

func (s *Service) BadCloseWithSectionSeparator(ctx context.Context) {
	conn := &closeable{}
	// ===================================================================
	_ = conn.Close() // want "discarding error from Close via blank assignment"
}

// =============================================================================
// Case 19: Close with blank comment - SHOULD FLAG
// =============================================================================

func (s *Service) BadCloseWithBlankComment(ctx context.Context) {
	conn := &closeable{}
	//
	_ = conn.Close() // want "discarding error from Close via blank assignment"
}
