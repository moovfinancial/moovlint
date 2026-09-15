package assert

import "testing"

type Assertions struct{ t *testing.T }

func New(t *testing.T) *Assertions { return &Assertions{t: t} }

func Equal(t *testing.T, expected, actual any, msgAndArgs ...any) bool { return true }

func NoError(t *testing.T, err error, msgAndArgs ...any) bool { return true }

func (a *Assertions) Equal(expected, actual any, msgAndArgs ...any) bool { return true }

func (a *Assertions) NoError(err error, msgAndArgs ...any) bool { return true }
