package midusage

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{}), "testdata/midusage")
}

func TestStringCompare(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(Config{StringCompare: true}), "testdata/midusage/stringcompare")
}
