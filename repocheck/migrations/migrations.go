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
	}

	return diags
}
