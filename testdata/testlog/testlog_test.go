package testlog

import (
	"fmt"
	"testing"
)

type logger struct{}

func (logger) Log(args ...any)                 {}
func (logger) Logf(format string, args ...any) {}

func TestLog(t *testing.T) {
	t.Log("value")        // want `t.Log in tests`
	t.Logf("value %d", 1) // want `t.Logf in tests`
	t.Errorf("value %d", 1)
	fmt.Sprint("value")
	logger{}.Log("value")
	logger{}.Logf("value %d", 1)
}

func BenchmarkLog(b *testing.B) {
	b.Logf("n=%d", b.N) // want `b.Logf in tests`
}

func FuzzLog(f *testing.F) {
	f.Log("seed") // want `f.Log in tests`
}

func helper(tb testing.TB) {
	tb.Helper()
	tb.Log("value") // want `tb.Log in tests`
}
