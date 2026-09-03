package logformat

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "logformat",
	Doc:  "detects %w verbs in Moov logger format strings; wrapping verbs are only valid in fmt.Errorf, use %v or %s",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			name := sel.Sel.Name
			if len(name) < 2 || !strings.HasSuffix(name, "f") {
				return true
			}
			if !moovutil.IsMoovLogPackage(moovutil.SelectorPackagePath(pass, sel)) {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if strings.Contains(lit.Value, "%w") {
				pass.Report(analysis.Diagnostic{
					Pos:     call.Pos(),
					Message: "%w is only valid in fmt.Errorf; logger format strings must use %v or %s",
				})
			}
			return true
		})
	}
	return nil, nil
}
