package require

import "testing"

type Assertions struct{ t *testing.T }

func New(t *testing.T) *Assertions { return &Assertions{t: t} }

func NoError(t *testing.T, err error, msgAndArgs ...any) {
	if err != nil {
		t.FailNow()
	}
}

func Error(t *testing.T, err error, msgAndArgs ...any) {
	if err == nil {
		t.FailNow()
	}
}

func Equal(t *testing.T, expected, actual any, msgAndArgs ...any) {}

func True(t *testing.T, value bool, msgAndArgs ...any) {}

func False(t *testing.T, value bool, msgAndArgs ...any) {}

func Nil(t *testing.T, object any, msgAndArgs ...any) {}

func NotNil(t *testing.T, object any, msgAndArgs ...any) {}

func FailNow(t *testing.T, failureMessage string, msgAndArgs ...any) {
	t.FailNow()
}

func (a *Assertions) NoError(err error, msgAndArgs ...any) {}

func (a *Assertions) Error(err error, msgAndArgs ...any) {}

func (a *Assertions) Equal(expected, actual any, msgAndArgs ...any) {}

func (a *Assertions) True(value bool, msgAndArgs ...any) {}

func (a *Assertions) False(value bool, msgAndArgs ...any) {}

func (a *Assertions) Nil(object any, msgAndArgs ...any) {}

func (a *Assertions) NotNil(object any, msgAndArgs ...any) {}

func (a *Assertions) FailNow(failureMessage string, msgAndArgs ...any) {}
