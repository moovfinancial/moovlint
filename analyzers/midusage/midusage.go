package midusage

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "midusage",
	Doc:  "detects mid.MustParseID usage outside test files",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if comparison, ok := n.(*ast.BinaryExpr); ok && (comparison.Op == token.EQL || comparison.Op == token.NEQ) &&
				(isMidID(pass.TypesInfo.TypeOf(comparison.X)) || isMidID(pass.TypesInfo.TypeOf(comparison.Y))) {
				pass.Report(analysis.Diagnostic{
					Pos:     comparison.Pos(),
					Message: "mid.ID values must use Equals instead of == or !=",
				})
			}

			if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel := selectorFromCall(call)
			if sel == nil {
				return true
			}
			if sel.Sel.Name != "MustParseID" {
				return true
			}
			pkgPath := moovutil.SelectorPackagePath(pass, sel)
			if !moovutil.IsMidPackage(pkgPath) {
				return true
			}
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				Message: "mid.MustParseID must not be used in production code; use mid.ParseID and handle the error",
			})
			return true
		})
	}
	return nil, nil
}

func isMidID(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Name() == "ID" && obj.Pkg() != nil && moovutil.IsMidPackage(obj.Pkg().Path())
}

func selectorFromCall(call *ast.CallExpr) *ast.SelectorExpr {
	fun := call.Fun
	if idx, ok := fun.(*ast.IndexExpr); ok {
		fun = idx.X
	}
	sel, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	return sel
}
