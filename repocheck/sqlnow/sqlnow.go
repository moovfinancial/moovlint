package sqlnow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/moovfinancial/moovlint/repocheck"
)

type SQLNowChecker struct{}

func (SQLNowChecker) Name() string { return "sqlnow" }

var nowRe = regexp.MustCompile(`(?i)\b(now\s*\(|current_timestamp|current_date|current_time|localtimestamp|localtime)`)

func (SQLNowChecker) Check(root string) ([]repocheck.Diagnostic, error) {
	files, err := findQuerySQLFiles(root)
	if err != nil {
		return nil, err
	}

	var diags []repocheck.Diagnostic
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "/*") {
				continue
			}
			match := nowRe.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			diags = append(diags, repocheck.Diagnostic{
				Path:  path,
				Line:  i + 1,
				Message: fmt.Sprintf("%s in a query file makes persisted values untestable; pass a timestamp parameter from the caller instead",
					strings.TrimSuffix(strings.ToLower(strings.TrimSpace(match[1])), "(")),
			})
		}
	}
	return diags, nil
}

func findQuerySQLFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "third_party" || name == ".git" || name == "node_modules" ||
				name == "migrations" || name == "testdata" || name == "testfixtures" || name == "golden" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".sql") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
