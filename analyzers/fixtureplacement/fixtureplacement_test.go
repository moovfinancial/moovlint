package fixtureplacement

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	dir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}
	analysistest.Run(t, dir, New(Config{Enabled: true}), "testdata/fixtureplacement", "testdata/fixtureplacement/fixtures")
	analysistest.Run(t, dir, New(Config{Enabled: true, FixturePackages: `/builders$`}), "testdata/fixtureplacement/builders")
}

func TestDisabled(t *testing.T) {
	if _, err := New(Config{}).Run(&analysis.Pass{}); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidConfig(t *testing.T) {
	if _, err := New(Config{Enabled: true, FixturePackages: "["}).Run(&analysis.Pass{}); err == nil {
		t.Fatal("invalid fixture package pattern must fail")
	}
}
