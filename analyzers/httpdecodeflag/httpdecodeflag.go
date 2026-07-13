package httpdecodeflag

import (
	"go/ast"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "httpdecodeflag",
	Doc:  "checks that HTTP request body decode errors are wrapped with errors.Flag(..., errors.NotSerializable)",
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
			ifStmt, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}

			decodeCall := findDecodeInIf(pass, ifStmt)
			if decodeCall == nil {
				return true
			}

			if !isBodyDecoderCall(pass, decodeCall) {
				return true
			}

			ast.Inspect(ifStmt.Body, func(n ast.Node) bool {
				ret, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				for _, expr := range ret.Results {
					if isFlaggedNotSerializable(pass, expr) {
						return true
					}
				}
				pass.Report(analysis.Diagnostic{
					Pos:     decodeCall.Pos(),
					Message: "request body decode error must be wrapped with errors.Flag(..., errors.NotSerializable) before returning",
				})
				return true
			})
			return true
		})
	}
	return nil, nil
}

func findDecodeInIf(pass *analysis.Pass, ifStmt *ast.IfStmt) *ast.CallExpr {
	if ifStmt.Init != nil {
		assign, ok := ifStmt.Init.(*ast.AssignStmt)
		if ok {
			for _, rhs := range assign.Rhs {
				if call, ok := rhs.(*ast.CallExpr); ok {
					if isDecodeCall(pass, call) {
						return call
					}
				}
			}
		}
	}
	if ifStmt.Cond != nil {
		if assign, ok := findDecodeInCondition(pass, ifStmt.Cond); ok {
			return assign
		}
	}
	return nil
}

func findDecodeInCondition(pass *analysis.Pass, cond ast.Expr) (*ast.CallExpr, bool) {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return nil, false
	}
	for _, side := range []ast.Expr{bin.X, bin.Y} {
		if call, ok := side.(*ast.CallExpr); ok {
			if isDecodeCall(pass, call) {
				return call, true
			}
		}
	}
	return nil, false
}

func isDecodeCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Decode"
}

func isBodyDecoderCall(pass *analysis.Pass, decodeCall *ast.CallExpr) bool {
	sel, ok := decodeCall.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Decode" {
		return false
	}
	inner, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	innerSel, ok := inner.Fun.(*ast.SelectorExpr)
	if !ok || innerSel.Sel.Name != "NewDecoder" {
		return false
	}
	pkgPath := moovutil.SelectorPackagePath(pass, innerSel)
	if pkgPath != "encoding/json" {
		return false
	}
	if len(inner.Args) < 1 {
		return false
	}
	arg, ok := inner.Args[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return arg.Sel != nil && arg.Sel.Name == "Body"
}

func isFlaggedNotSerializable(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Flag" {
		return false
	}
	if !moovutil.IsErrorsPackage(moovutil.SelectorPackagePath(pass, sel)) {
		return false
	}
	for _, arg := range call.Args {
		if isNotSerializableIdent(pass, arg) {
			return true
		}
	}
	return false
}

func isNotSerializableIdent(pass *analysis.Pass, expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "NotSerializable" {
		return false
	}
	return moovutil.IsErrorsPackage(moovutil.SelectorPackagePath(pass, sel))
}
