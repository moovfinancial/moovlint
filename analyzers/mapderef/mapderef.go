package mapderef

import (
	"go/ast"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "mapderef",
	Doc:  "detects m[k].Field dereferences on maps of pointers or interfaces without a comma-ok check; a missing key yields nil",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.SelectorExpr:
				if idx, ok := node.X.(*ast.IndexExpr); ok && isNilableElem(pass, idx) {
					report(pass, idx)
				}
			case *ast.StarExpr:
				if idx, ok := node.X.(*ast.IndexExpr); ok && isNilableElem(pass, idx) {
					report(pass, idx)
				}
			}
			return true
		})
	}
	return nil, nil
}

func isNilableElem(pass *analysis.Pass, idx *ast.IndexExpr) bool {
	t := pass.TypesInfo.TypeOf(idx)
	if t == nil {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return true
	}
	return false
}

func report(pass *analysis.Pass, idx *ast.IndexExpr) {
	pass.Report(analysis.Diagnostic{
		Pos:     idx.Pos(),
		Message: "map lookup is dereferenced without a comma-ok check; a missing key yields a nil value, use v, ok := m[k]",
	})
}
