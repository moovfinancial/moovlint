package timeinject

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{}), "testdata/timeinject")
}

func TestTimers(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{Timers: true}), "testdata/timeinject/timers")
}
