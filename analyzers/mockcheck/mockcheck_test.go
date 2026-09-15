package mockcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "testdata/mockcheck")
}

func TestAllowedInterfaces(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{AllowInterfaces: `\.AllowedBoundary$`}), "testdata/mockcheck/allow")
}

func TestInvalidConfig(t *testing.T) {
	if _, err := New(Config{AllowInterfaces: "["}).Run(&analysis.Pass{}); err == nil {
		t.Fatal("invalid interface pattern must fail")
	}
}
