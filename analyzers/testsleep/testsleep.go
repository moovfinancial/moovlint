package testsleep

import (
	"go/ast"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "testsleep",
	Doc:  "detects time.Sleep used for synchronization in test files; use require.Eventually or an injected clock",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if !moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Sleep" {
				return true
			}
			if moovutil.SelectorPackagePath(pass, sel) != "time" {
				return true
			}
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				Message: "time.Sleep in tests causes flaky synchronization; use require.Eventually or an injected clock (nolint with a comment if the sleep is the behavior under test)",
			})
			return true
		})
	}
	return nil, nil
}
