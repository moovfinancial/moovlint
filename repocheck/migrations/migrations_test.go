package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tmp := t.TempDir()
	migDir := filepath.Join(tmp, "migrations")
	if err := os.MkdirAll(migDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"001_create_table.up.postgres.sql":   "CREATE TABLE users (id TEXT NOT NULL, name TEXT);",
		"001_create_table.down.postgres.sql": "DROP TABLE users;",
		"002_add_col.up.postgres.sql":        "ALTER TABLE users ADD COLUMN email TEXT;",
		"003_bad_name.up.postgres.sql":       "ALTER TABLE users ADD COLUMN status TEXT NOT NULL;",
		"005_ifnotexists.up.postgres.sql":    "CREATE TABLE IF NOT EXISTS logs (id TEXT);",
		"006_rename.up.postgres.sql":         "ALTER TABLE users RENAME COLUMN name TO full_name;",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(migDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	c := MigrationsChecker{}
	diags, err := c.Check(tmp)
	if err != nil {
		t.Fatal(err)
	}

	wantSubs := []string{"NOT NULL", "IF NOT EXISTS", "rolling-safe", "out of order"}
	for _, sub := range wantSubs {
		found := false
		for _, d := range diags {
			if strings.Contains(d.Message, sub) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected diagnostic containing %q, got %d diagnostics: %v", sub, len(diags), diags)
		}
	}
}
