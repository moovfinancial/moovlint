package testifysubtestassertion

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, filepath.Join(analysistest.TestData(), "testifysubtestassertion"), Analyzer)
}
