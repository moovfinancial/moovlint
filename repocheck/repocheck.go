package repocheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Diagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

type Checker interface {
	Name() string
	Check(root string) ([]Diagnostic, error)
}

func RunAll(root string, checkers []Checker) ([]Diagnostic, error) {
	var diags []Diagnostic
	for _, c := range checkers {
		d, err := c.Check(root)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.Name(), err)
		}
		diags = append(diags, d...)
	}
	return diags, nil
}

func PrintDiagnostics(diags []Diagnostic) {
	for _, d := range diags {
		loc := d.Path
		if d.Line > 0 {
			loc = fmt.Sprintf("%s:%d", d.Path, d.Line)
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", loc, d.Message)
	}
}

func findFiles(root, suffix string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "third_party" || name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, suffix) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func FindGoFiles(root string) ([]string, error) {
	return findFiles(root, ".go")
}
