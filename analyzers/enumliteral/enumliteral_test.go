package enumliteral

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{Enabled: true}), "testdata/enumliteral", "testdata/enumliteral/status")
}

func TestDisabled(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{}), "testdata/enumliteral/disabled")
}
