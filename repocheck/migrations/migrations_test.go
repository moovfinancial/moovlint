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

func TestEmptyStringGuards(t *testing.T) {
	tmp := t.TempDir()
	migDir := filepath.Join(tmp, "migrations")
	if err := os.MkdirAll(migDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"001_keys.up.postgres.sql": `CREATE TABLE runs (
	run_id TEXT NOT NULL,
	base_sha TEXT NOT NULL,
	description TEXT NOT NULL,
	external_ref TEXT NOT NULL CHECK (external_ref <> '')
);`,
		"002_guarded.up.postgres.sql": `CREATE TABLE events (
	event_id TEXT NOT NULL CHECK (event_id <> ''),
	kind TEXT
);`,
		"003_nonkey.up.postgres.sql": `CREATE TABLE notes (
	body TEXT NOT NULL,
	label TEXT NOT NULL
);`,
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

	wantCols := []string{"run_id", "base_sha"}
	notWant := []string{"description", "external_ref", "event_id", "body", "label"}
	for _, col := range wantCols {
		found := false
		for _, d := range diags {
			if strings.Contains(d.Message, "column "+col+" is TEXT NOT NULL") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected empty-string CHECK diagnostic for %s, got %d diagnostics: %v", col, len(diags), diags)
		}
	}
	for _, col := range notWant {
		for _, d := range diags {
			if strings.Contains(d.Message, "column "+col+" ") {
				t.Errorf("column %s should not be flagged: %s", col, d.Message)
			}
		}
	}
}
