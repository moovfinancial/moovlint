package structtags

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tmp := t.TempDir()
	src := "package test\n\ntype User struct {\n" +
		"\tAccountID  string `json:\"accountID\"`\n" +
		"\tUserName   string `json:\"user_name\"`\n" +
		"\tCreatedOn  string `json:\"created\"`\n" +
		"\tGoodField  string `json:\"goodField\"`\n" +
	 "}\n"

	if err := os.WriteFile(filepath.Join(tmp, "model.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	c := StructTagsChecker{}
	diags, err := c.Check(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(diags) < 2 {
		t.Fatalf("expected at least 2 diagnostics, got %d: %v", len(diags), diags)
	}

	wantSubs := []string{"camelCase", "timestamp"}
	for _, sub := range wantSubs {
		found := false
		for _, d := range diags {
			if strings.Contains(d.Message, sub) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected diagnostic containing %q, got %d diagnostics", sub, len(diags))
		}
	}
}
