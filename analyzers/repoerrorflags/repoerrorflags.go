package repoerrorflags

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

type Config struct {
	Enabled bool `json:"enabled"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "repoerrorflags",
		Doc:  "checks that repository methods flag expected database errors with the correct errors.Flag (advisory, opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable advisory repository error flag checks")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		return run(pass)
	}
	return a
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

			if !propagatesUnflagged(pass, ifStmt, expectedFlag) {
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

var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// propagatesUnflagged reports whether the branch returns the database error
// without the expected flag. A branch that recovers (returns nil, or returns
// an error that does not come from the checked error) is not a finding.
func propagatesUnflagged(pass *analysis.Pass, ifStmt *ast.IfStmt, flag string) bool {
	checked := make(map[types.Object]bool)
	ast.Inspect(ifStmt.Cond, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if v, ok := pass.TypesInfo.ObjectOf(id).(*types.Var); ok && types.Implements(v.Type(), errorIface) {
				checked[v] = true
			}
		}
		return true
	})

	found := false
	ast.Inspect(ifStmt.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok || found {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, res := range ret.Results {
			t := pass.TypesInfo.TypeOf(res)
			if pass.TypesInfo.Types[res].IsNil() || t == nil || !types.Implements(t, errorIface) {
				continue
			}
			if hasFlagCall(pass, res, flag) {
				continue
			}
			if len(checked) == 0 || references(pass, res, checked) {
				found = true
			}
		}
		return true
	})
	return found
}

func references(pass *analysis.Pass, expr ast.Expr, objs map[types.Object]bool) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && objs[pass.TypesInfo.ObjectOf(id)] {
			found = true
		}
		return !found
	})
	return found
}

func detectExpectedFlag(pass *analysis.Pass, ifStmt *ast.IfStmt) string {
	if call, ok := ast.Unparen(ifStmt.Cond).(*ast.CallExpr); ok && isSQLErrNoRowsCheck(pass, call) {
		return "NotFound"
	}
	bin, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.EQL {
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
