package midusage

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

// Config holds opt-in extensions. The base checks always run.
type Config struct {
	// StringCompare also flags a.String() == b.String() on two mid.IDs and
	// id.String() == "" zero checks.
	StringCompare bool `json:"stringcompare"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "midusage",
		Doc:  "detects mid.MustParseID usage outside test files",
	}
	a.Flags.BoolVar(&cfg.StringCompare, "stringcompare", cfg.StringCompare, "also flag mid.ID comparisons through String()")
	a.Run = func(pass *analysis.Pass) (any, error) { return run(pass, cfg) }
	return a
}

func run(pass *analysis.Pass, cfg Config) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if comparison, ok := n.(*ast.BinaryExpr); ok && (comparison.Op == token.EQL || comparison.Op == token.NEQ) {
				if isMidID(pass.TypesInfo.TypeOf(comparison.X)) || isMidID(pass.TypesInfo.TypeOf(comparison.Y)) {
					pass.Report(analysis.Diagnostic{
						Pos:     comparison.Pos(),
						Message: "mid.ID values must use Equals instead of == or !=",
					})
				} else if cfg.StringCompare {
					checkStringCompare(pass, comparison)
				}
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

func checkStringCompare(pass *analysis.Pass, cmp *ast.BinaryExpr) {
	x, y := isIDString(pass, cmp.X), isIDString(pass, cmp.Y)
	switch {
	case x && y:
		pass.Reportf(cmp.Pos(), "compare mid.ID values with Equals, not through String()")
	case x && isEmptyString(pass, cmp.Y), y && isEmptyString(pass, cmp.X):
		pass.Reportf(cmp.Pos(), "use IsEmpty() on the mid.ID instead of comparing String() with \"\"")
	}
}

func isIDString(pass *analysis.Pass, e ast.Expr) bool {
	call, ok := ast.Unparen(e).(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "String" {
		return false
	}
	t := pass.TypesInfo.TypeOf(sel.X)
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	return isMidID(t)
}

func isEmptyString(pass *analysis.Pass, e ast.Expr) bool {
	tv := pass.TypesInfo.Types[e]
	return tv.Value != nil && tv.Value.ExactString() == `""`
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
