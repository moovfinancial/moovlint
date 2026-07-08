// Package thelper implements an analyzer that requires test helpers to call
// t.Helper() before any t.Fatal/t.Errorf/require.X/assert.X usage.
//
// Without t.Helper(), failure report file:line points at the helper instead
// of the caller, which obscures the actual failing test from the developer.
package thelper

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "thelper",
	Doc:  "requires test helpers that call t.Fatal/t.Error/require/assert to call t.Helper() as their first statement",
	Run:  run,
}

// testifyPackagePaths marks imports that signal usage of testify
// assertion-style calls inside the same file.
var testifyPackagePaths = []string{
	"github.com/stretchr/testify/require",
	"github.com/stretchr/testify/assert",
	"github.com/stretchr/testify/suite",
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if !isTestFile(pass, file) {
			continue
		}

		testifyInFile := fileHasTestifyImport(file)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if !takesTestingT(fn) {
				continue
			}

			callsHelper := false
			hasFatalCall := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				recv, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				if recv.Name == "t" {
					switch sel.Sel.Name {
					case "Helper":
						callsHelper = true
					case "Fatal", "Fatalf", "FailNow", "Error", "Errorf", "Fail":
						hasFatalCall = true
					}
					return true
				}

				if testifyInFile {
					switch recv.Name {
					case "require", "assert":
						hasFatalCall = true
					}
				}
				return true
			})

			if callsHelper || !hasFatalCall {
				continue
			}

			pass.Report(analysis.Diagnostic{
				Pos:     fn.Pos(),
				End:     fn.Type.Pos(),
				Message: "test helper '" + fn.Name.Name + "' performs t.Fatal/t.Error/require.X/assert.X but does not call t.Helper(); add t.Helper() as the first statement so failure reports point at the caller",
			})
		}
	}
	return nil, nil
}

func isTestFile(pass *analysis.Pass, file *ast.File) bool {
	return strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go")
}

func takesTestingT(fn *ast.FuncDecl) bool {
	params := fn.Type.Params
	if params == nil || len(params.List) == 0 {
		return false
	}
	first := params.List[0]
	if len(first.Names) == 0 {
		return false
	}
	if first.Names[0].Name != "t" {
		return false
	}
	star, ok := first.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "T" {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	return ok && pkgIdent.Name == "testing"
}

func fileHasTestifyImport(file *ast.File) bool {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		for _, allowed := range testifyPackagePaths {
			if path == allowed {
				return true
			}
		}
	}
	return false
}
