package blankdiscard

import "io"

// ErrorReturner demonstrates functions that return errors
type ErrorReturner struct{}

func (e *ErrorReturner) ReturnsError() error {
	return nil
}

func (e *ErrorReturner) ReturnsIntAndError() (int, error) {
	return 0, nil
}

func (e *ErrorReturner) ReturnsInt() int {
	return 0
}

func (e *ErrorReturner) ReturnsTwoInts() (int, int) {
	return 0, 0
}

// Closer implements io.Closer
type Closer struct{}

func (c *Closer) Close() error {
	return nil
}

type Service struct{}

// Single error discard - should be flagged
func (s *Service) BadSingleDiscard() {
	e := &ErrorReturner{}
	_ = e.ReturnsError() // want "discarding error return value; handle the error or use an explanatory comment if intentional"
}

// Multi-value discard where one is error - should be flagged
func (s *Service) BadMultiDiscard() {
	e := &ErrorReturner{}
	_, _ = e.ReturnsIntAndError() // want "discarding error return value; handle the error or use an explanatory comment if intentional"
}

// Close without comment - should be flagged
func (s *Service) BadCloseNoComment() {
	c := &Closer{}
	_ = c.Close() // want "discarding error return value; handle the error or use an explanatory comment if intentional"
}

// Close with inline comment - should NOT be flagged
func (s *Service) GoodCloseInlineComment() {
	c := &Closer{}
	_ = c.Close() // connection already closed, ignore error
}

// Close with preceding comment - should NOT be flagged
func (s *Service) GoodClosePrecedingComment() {
	c := &Closer{}
	// Best effort cleanup, error is expected
	_ = c.Close()
}

// Non-error discard - should NOT be flagged
func (s *Service) GoodNonErrorDiscard() {
	e := &ErrorReturner{}
	_ = e.ReturnsInt()
}

// Multi-value non-error discard - should NOT be flagged
func (s *Service) GoodMultiNonErrorDiscard() {
	e := &ErrorReturner{}
	_, _ = e.ReturnsTwoInts()
}

// Named error variable - should NOT be flagged
func (s *Service) GoodNamedError() {
	e := &ErrorReturner{}
	_, err := e.ReturnsIntAndError()
	_ = err
}

// Error assigned to named variable - should NOT be flagged
func (s *Service) GoodHandledError() {
	e := &ErrorReturner{}
	err := e.ReturnsError()
	_ = err
}

// Close() from io.Closer interface with comment - should NOT be flagged
func (s *Service) GoodIOCloserWithComment(rc io.ReadCloser) {
	_ = rc.Close() // ignoring close error
}

// Close() from io.Closer interface without comment - should be flagged
func (s *Service) BadIOCloserNoComment(rc io.ReadCloser) {
	_ = rc.Close() // want "discarding error return value; handle the error or use an explanatory comment if intentional"
}

// Single error discard with explanatory inline comment - should NOT be flagged
func (s *Service) GoodSingleDiscardWithComment() {
	e := &ErrorReturner{}
	_ = e.ReturnsError() // error intentionally ignored, retrying would cause loop
}
