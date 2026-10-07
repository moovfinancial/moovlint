package testlog

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{Enabled: true}), "testdata/testlog")
}

func TestDisabled(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{}), "testdata/testlog/disabled")
}
