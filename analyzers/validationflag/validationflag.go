package validationflag

import (
	"go/ast"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "validationflag",
	Doc:  "checks that Validate() error methods wrap mvalidation.ValidateStruct returns with errors.Flag(..., errors.NotValid)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsMoovPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			if fn.Name == nil || fn.Name.Name != "Validate" {
				return true
			}
			if !isErrorReturning(fn) {
				return true
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				ret, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				for _, expr := range ret.Results {
					if isBareValidateStructCall(pass, expr) {
						pass.Report(analysis.Diagnostic{
							Pos:     ret.Pos(),
							Message: "mvalidation.ValidateStruct result must be wrapped with errors.Flag(err, errors.NotValid) before returning from Validate()",
						})
					}
				}
				return true
			})
			return true
		})
	}
	return nil, nil
}

func isErrorReturning(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Results == nil {
		return false
	}
	for _, result := range fn.Type.Results.List {
		ident, ok := result.Type.(*ast.Ident)
		if ok && ident.Name == "error" {
			return true
		}
	}
	return false
}

func isBareValidateStructCall(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "ValidateStruct" {
		return false
	}
	pkgPath := moovutil.SelectorPackagePath(pass, sel)
	return moovutil.IsMValidationPackage(pkgPath)
}
