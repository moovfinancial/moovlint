package sqlnow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tmp := t.TempDir()
	queryDir := filepath.Join(tmp, "internal", "sqlc", "queries")
	if err := os.MkdirAll(queryDir, 0755); err != nil {
		t.Fatal(err)
	}
	migDir := filepath.Join(tmp, "migrations")
	if err := os.MkdirAll(migDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(queryDir, "effects.sql"): "-- name: CompleteEffect :exec\nUPDATE effects SET completed_on = now() WHERE id = $1;\n",
		filepath.Join(queryDir, "cursors.sql"): "INSERT INTO cursors (id, updated_at) VALUES ($1, CURRENT_TIMESTAMP);\n",
		filepath.Join(queryDir, "clean.sql"):   "-- now() is only mentioned in this comment\nSELECT id FROM effects;\n",
		filepath.Join(migDir, "001_init.sql"):  "CREATE TABLE t (x TIMESTAMP DEFAULT NOW());\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	c := SQLNowChecker{}
	diags, err := c.Check(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics (migrations and comments excluded), got %d: %v", len(diags), diags)
	}
	for _, d := range diags {
		if !strings.Contains(d.Message, "untestable") {
			t.Errorf("unexpected message: %s", d.Message)
		}
		if d.Line == 0 {
			t.Errorf("diagnostic missing line number: %+v", d)
		}
	}
}
