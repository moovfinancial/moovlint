package modelplacement

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
	analysistest.Run(t, dir, New(Config{Enabled: true}), "testdata/modelplacement")
	analysistest.Run(t, dir, New(Config{
		Enabled: true, ModelFiles: `^types\.go$`, ImplementationFiles: `^handler\.go$`, ModelTypes: `DTO$`,
	}), "testdata/modelplacement/custom")
}

func TestDisabled(t *testing.T) {
	if _, err := New(Config{}).Run(&analysis.Pass{}); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidConfig(t *testing.T) {
	for _, cfg := range []Config{
		{Enabled: true, ModelFiles: "["},
		{Enabled: true, ImplementationFiles: "["},
		{Enabled: true, ModelTypes: "["},
	} {
		if _, err := New(cfg).Run(&analysis.Pass{}); err == nil {
			t.Errorf("invalid config must fail: %+v", cfg)
		}
	}
}
