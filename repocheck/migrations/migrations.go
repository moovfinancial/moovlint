package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/moovfinancial/moovlint/repocheck"
)

type MigrationsChecker struct{}

func (MigrationsChecker) Name() string { return "migrations" }

func (MigrationsChecker) Check(root string) ([]repocheck.Diagnostic, error) {
	var diags []repocheck.Diagnostic

	migrationDirs, err := findMigrationDirs(root)
	if err != nil {
		return nil, err
	}
	if len(migrationDirs) == 0 {
		return nil, nil
	}

	for _, dir := range migrationDirs {
		d := checkMigrationDir(dir)
		diags = append(diags, d...)
	}
	return diags, nil
}

func findMigrationDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		name := info.Name()
		if name == "vendor" || name == "third_party" || name == ".git" || name == "node_modules" {
			return filepath.SkipDir
		}
		if name == "migrations" {
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs, err
}

var migrationNameRe = regexp.MustCompile(`^(\d+)[-_](.*)\.(up|down)\.(postgres\.)?sql$`)

func checkMigrationDir(dir string) []repocheck.Diagnostic {
	var diags []repocheck.Diagnostic

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	expectedSeq := 1
	for _, name := range names {
		m := migrationNameRe.FindStringSubmatch(name)
		if m == nil {
			diags = append(diags, repocheck.Diagnostic{
				Path:    filepath.Join(dir, name),
				Message: fmt.Sprintf("migration file %q does not match expected pattern NNN_description.(up|down).[postgres.]sql", name),
			})
			continue
		}

		seq, _ := strconv.Atoi(m[1])
		if seq != expectedSeq {
			diags = append(diags, repocheck.Diagnostic{
				Path:    filepath.Join(dir, name),
				Message: fmt.Sprintf("migration sequence number %d is out of order; expected %d", seq, expectedSeq),
			})
		}
		if seq == expectedSeq {
			expectedSeq++
		}
		if seq > expectedSeq {
			expectedSeq = seq + 1
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		sql := string(data)

		if strings.Contains(strings.ToUpper(sql), "IF NOT EXISTS") {
			diags = append(diags, repocheck.Diagnostic{
				Path:    filepath.Join(dir, name),
				Message: "migrations must not use IF NOT EXISTS; migrations are applied once in order",
			})
		}

		if strings.Contains(strings.ToUpper(sql), "ALTER TABLE") && strings.Contains(strings.ToUpper(sql), "RENAME") {
			diags = append(diags, repocheck.Diagnostic{
				Path:    filepath.Join(dir, name),
				Message: "direct column/table renames are not rolling-safe; add a new column, copy data, then drop in a later migration",
			})
		}

		upperSQL := strings.ToUpper(sql)
		if strings.Contains(upperSQL, "ADD COLUMN") && strings.Contains(upperSQL, "NOT NULL") && !strings.Contains(upperSQL, "DEFAULT") {
			diags = append(diags, repocheck.Diagnostic{
				Path:    filepath.Join(dir, name),
				Message: "adding NOT NULL column without DEFAULT is not rolling-safe; add as nullable first, backfill, then add NOT NULL in a later migration",
			})
		}

		diags = append(diags, checkEmptyStringGuards(sql, filepath.Join(dir, name))...)
	}

	return diags
}

var keyColumnSuffixes = []string{"id", "key", "sha", "code", "ref", "uuid", "hash", "token", "type", "external"}

var textNotNullRe = regexp.MustCompile(`(?i)(?:[(,]|add\s+column\s+)\s*"?([a-zA-Z_][a-zA-Z0-9_]*)"?\s+(?:TEXT|VARCHAR(?:\s*\(\d+\))?|CHARACTER\s+VARYING(?:\s*\(\d+\))?|CHAR(?:\s*\(\d+\))?)\s+[^;,()]*NOT\s+NULL`)

// checkEmptyStringGuards flags identifier-like TEXT NOT NULL columns that lack
// an empty-string CHECK constraint, matching the peer key-column convention.
func checkEmptyStringGuards(sql, path string) []repocheck.Diagnostic {
	var diags []repocheck.Diagnostic
	for _, m := range textNotNullRe.FindAllStringSubmatch(sql, -1) {
		col := m[1]
		if !isKeyColumn(col) {
			continue
		}
		if hasEmptyStringGuard(sql, col) {
			continue
		}
		diags = append(diags, repocheck.Diagnostic{
			Path:    path,
			Line:    lineOf(sql, m[0]),
			Message: fmt.Sprintf("column %s is TEXT NOT NULL without an empty-string CHECK constraint; peer key columns require CHECK (%s <> '')", col, col),
		})
	}
	return diags
}

func isKeyColumn(col string) bool {
	lower := strings.ToLower(col)
	for _, suffix := range keyColumnSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func hasEmptyStringGuard(sql, col string) bool {
	patterns := []string{
		`(?i)check\s*\(\s*"?` + regexp.QuoteMeta(col) + `"?\s*(?:<>|!=|>|<)\s*''`,
		`(?i)check\s*\(\s*''\s*(?:<>|!=|>|<)\s*"?` + regexp.QuoteMeta(col) + `"?\s*\)`,
		`(?i)check\s*\(\s*length\s*\(\s*"?` + regexp.QuoteMeta(col) + `"?\s*\)\s*>\s*0`,
	}
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err == nil && re.MatchString(sql) {
			return true
		}
	}
	return false
}

func lineOf(sql, match string) int {
	idx := strings.Index(sql, match)
	if idx < 0 {
		return 0
	}
	return strings.Count(sql[:idx], "\n") + 1
}
