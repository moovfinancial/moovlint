package subtestassert

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{}), "testdata/subtestassert")
}

func TestOuterT(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{OuterT: true}), "testdata/subtestassert/outert")
}
