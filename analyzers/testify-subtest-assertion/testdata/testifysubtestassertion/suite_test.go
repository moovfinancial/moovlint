package testifysubtestassertion

import "testing"

type fakeAssert struct{ t *testing.T }

func (a *fakeAssert) Equal(expected, got any, msgAndArgs ...any) bool {
	a.t.Helper()
	return expected == got
}

type fakeRequire struct{ t *testing.T }

func (r *fakeRequire) NoError(err error, msgAndArgs ...any) bool {
	r.t.Helper()
	return err == nil
}

type fakeSuite struct {
	*testing.T
	Assert  *fakeAssert
	Require *fakeRequire
}

func newSuite(t *testing.T) *fakeSuite {
	return &fakeSuite{T: t, Assert: &fakeAssert{t: t}, Require: &fakeRequire{t: t}}
}

func (s *fakeSuite) TestSubtestMisuseFlags() {
	s.Run("nested-misuse", func(t *testing.T) {
		s.Assert.Equal(1, 1)    // want `suite "Assert" called inside t.Run subtest closure`
		s.Require.NoError(nil) // want `suite "Require" called inside t.Run subtest closure`
	})
}

func (s *fakeSuite) TestSubtestClean() {
	s.Run("nested-clean", func(t *testing.T) {
		a := &fakeAssert{t: t}
		r := &fakeRequire{t: t}
		a.Equal(1, 1)
		r.NoError(nil)
	})
}

func (s *fakeSuite) TestNotInsideSubtest() {
	s.Assert.Equal(1, 1)   // OK: not inside t.Run
	s.Require.NoError(nil) // OK: not inside t.Run
}

func TestSuiteDriver(t *testing.T) {
	s := newSuite(t)
	s.TestSubtestMisuseFlags()
	s.TestSubtestClean()
	s.TestNotInsideSubtest()
}
