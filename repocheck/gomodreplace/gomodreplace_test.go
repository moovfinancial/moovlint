package gomodreplace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tmp := t.TempDir()
	goMod := `module example.com/service

go 1.26.4

require github.com/moovfinancial/go-libs v1.2.3

replace github.com/foo/bar => ../bar

replace github.com/baz/qux => ../qux // TODO(LINEAR-42) revert after upstream release

// TODO(LINEAR-77) local dev override
replace github.com/a/b => ../b

replace (
	github.com/c/d => ../d
	github.com/e/f => ../f // ENG-9 temporary fork
	github.com/g/h => ../h
)
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	c := GoModReplaceChecker{}
	diags, err := c.Check(tmp)
	if err != nil {
		t.Fatal(err)
	}

	// flagged: github.com/foo/bar (no comment), github.com/c/d, github.com/g/h
	if len(diags) != 3 {
		t.Fatalf("expected 3 diagnostics, got %d: %v", len(diags), diags)
	}
	for _, d := range diags {
		if !strings.Contains(d.Message, "tracking ticket") {
			t.Errorf("unexpected message: %s", d.Message)
		}
		if d.Line == 0 {
			t.Errorf("diagnostic missing line number: %+v", d)
		}
	}
	flaggedRules := []string{}
	for _, d := range diags {
		flaggedRules = append(flaggedRules, d.Message)
	}
	joined := strings.Join(flaggedRules, "\n")
	for _, must := range []string{"github.com/foo/bar", "github.com/c/d", "github.com/g/h"} {
		if !strings.Contains(joined, must) {
			t.Errorf("expected rule %s to be flagged, got: %s", must, joined)
		}
	}
	for _, mustNot := range []string{"github.com/baz/qux", "github.com/a/b", "github.com/e/f"} {
		if strings.Contains(joined, mustNot) {
			t.Errorf("rule %s should not be flagged, got: %s", mustNot, joined)
		}
	}
}

func TestCheckNoGoMod(t *testing.T) {
	c := GoModReplaceChecker{}
	diags, err := c.Check(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}
