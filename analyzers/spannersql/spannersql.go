package spannersql

import (
	"go/ast"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

const spannerPkg = "cloud.google.com/go/spanner"

type Config struct {
	Enabled bool `json:"enabled"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "spannersql",
		Doc:  "detects Spanner SQL built with fmt.Sprintf from non-constant values (opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable Spanner SQL checks")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		return run(pass)
	}
	return a
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsMoovPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Pos()).Filename) {
			continue
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				checkFunc(pass, fn)
			}
		}
	}
	return nil, nil
}

func checkFunc(pass *analysis.Pass, fn *ast.FuncDecl) {
	// Local variables assigned from a Sprintf with a non-constant argument.
	tainted := map[types.Object]bool{}
	mark := func(lhs []*ast.Ident, rhs []ast.Expr) {
		if len(lhs) != len(rhs) {
			return
		}
		for i, id := range lhs {
			if id == nil {
				continue
			}
			obj := pass.TypesInfo.ObjectOf(id)
			if obj != nil && obj.Pos() >= fn.Pos() && obj.Pos() < fn.End() && isUnsafeSprintf(pass, rhs[i]) {
				tainted[obj] = true
			}
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			var ids []*ast.Ident
			for _, l := range n.Lhs {
				id, _ := l.(*ast.Ident)
				ids = append(ids, id)
			}
			mark(ids, n.Rhs)
		case *ast.ValueSpec:
			mark(n.Names, n.Values)
		}
		return true
	})

	check := func(e ast.Expr) {
		if id, ok := e.(*ast.Ident); ok && tainted[pass.TypesInfo.ObjectOf(id)] || isUnsafeSprintf(pass, e) {
			pass.Reportf(e.Pos(), "Spanner SQL built with fmt.Sprintf from a non-constant value; pass values in Statement.Params")
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CompositeLit:
			if !isSpannerObj(typeObj(pass.TypesInfo.TypeOf(n)), "Statement") || len(n.Elts) == 0 {
				return true
			}
			for _, elt := range n.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "SQL" {
						check(kv.Value)
					}
				}
			}
			if _, ok := n.Elts[0].(*ast.KeyValueExpr); !ok {
				check(n.Elts[0])
			}
		case *ast.CallExpr:
			if isSpannerObj(calleeObj(pass, n), "NewStatement") && len(n.Args) == 1 {
				check(n.Args[0])
			}
		}
		return true
	})
}

func isUnsafeSprintf(pass *analysis.Pass, e ast.Expr) bool {
	call, ok := ast.Unparen(e).(*ast.CallExpr)
	if !ok {
		return false
	}
	obj := calleeObj(pass, call)
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != "fmt" || obj.Name() != "Sprintf" {
		return false
	}
	for _, arg := range call.Args[1:] {
		if pass.TypesInfo.Types[arg].Value == nil {
			return true
		}
	}
	return false
}

func calleeObj(pass *analysis.Pass, call *ast.CallExpr) types.Object {
	switch f := ast.Unparen(call.Fun).(type) {
	case *ast.SelectorExpr:
		return pass.TypesInfo.Uses[f.Sel]
	case *ast.Ident:
		return pass.TypesInfo.Uses[f]
	}
	return nil
}

func typeObj(t types.Type) types.Object {
	if named, ok := types.Unalias(t).(*types.Named); ok {
		return named.Obj()
	}
	return nil
}

func isSpannerObj(obj types.Object, name string) bool {
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == spannerPkg && obj.Name() == name
}
