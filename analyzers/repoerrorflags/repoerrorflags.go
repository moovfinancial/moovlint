package repoerrorflags

import (
	"fmt"
	"go/ast"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "repoerrorflags",
	Doc:  "checks that repository methods flag expected database errors with the correct errors.Flag",
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
			if !ok || ifStmt.Body == nil {
				return true
			}

			expectedFlag := detectExpectedFlag(pass, ifStmt)
			if expectedFlag == "" {
				return true
			}

			if hasFlagInBody(pass, ifStmt.Body, expectedFlag) {
				return true
			}

			pass.Report(analysis.Diagnostic{
				Pos:     ifStmt.Pos(),
				Message: fmt.Sprintf("database error check should be flagged with errors.%s in this branch", expectedFlag),
			})
			return true
		})
	}
	return nil, nil
}

func detectExpectedFlag(pass *analysis.Pass, ifStmt *ast.IfStmt) string {
	bin, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return ""
	}
	for _, side := range []ast.Expr{bin.X, bin.Y} {
		call, ok := side.(*ast.CallExpr)
		if !ok {
			continue
		}
		if isSpannerErrCode(call) {
			otherSide := bin.X
			if side == bin.X {
				otherSide = bin.Y
			}
			if code := extractCodeConst(pass, otherSide); code != "" {
				switch code {
				case "AlreadyExists":
					return "NotUnique"
				case "NotFound":
					return "NotFound"
				}
			}
		}
		if isSQLErrNoRowsCheck(pass, call) {
			return "NotFound"
		}
	}
	return ""
}

func isSpannerErrCode(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "ErrCode"
}

func extractCodeConst(pass *analysis.Pass, expr ast.Expr) string {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return sel.Sel.Name
}

func isSQLErrNoRowsCheck(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Is" {
		return false
	}
	pkgPath := moovutil.SelectorPackagePath(pass, sel)
	if pkgPath != "errors" && !moovutil.IsErrorsPackage(pkgPath) {
		return false
	}
	if len(call.Args) < 2 {
		return false
	}
	arg, ok := call.Args[1].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return arg.Sel.Name == "ErrNoRows"
}

func hasFlagInBody(pass *analysis.Pass, body *ast.BlockStmt, expectedFlag string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, expr := range ret.Results {
			if hasFlagCall(pass, expr, expectedFlag) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func hasFlagCall(pass *analysis.Pass, expr ast.Expr, flagName string) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	found := false
	ast.Inspect(call, func(n ast.Node) bool {
		if found {
			return false
		}
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Flag" {
			return true
		}
		if !moovutil.IsErrorsPackage(moovutil.SelectorPackagePath(pass, sel)) {
			return true
		}
		for _, arg := range c.Args {
			if isFlagConst(pass, arg, flagName) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func isFlagConst(pass *analysis.Pass, expr ast.Expr, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != name {
		return false
	}
	return moovutil.IsErrorsPackage(moovutil.SelectorPackagePath(pass, sel))
}
