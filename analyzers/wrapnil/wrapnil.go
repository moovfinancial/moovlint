package wrapnil

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

type Config struct {
	Enabled bool `json:"enabled"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "wrapnil",
		Doc:  "detects fmt.Errorf %w of an error that can be nil because the if condition is an || chain (opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable wrap-nil checks")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		return run(pass)
	}
	return a
}

var errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsMoovPackage(pass.Pkg.Path()) {
		return nil, nil
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			ifStmt, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			operands := orOperands(ast.Unparen(ifStmt.Cond))
			for _, op := range operands {
				if obj := nonNilErr(pass, op); obj != nil && len(operands) > 1 {
					checkBody(pass, ifStmt.Body, obj)
				}
			}
			return true
		})
	}
	return nil, nil
}

func orOperands(e ast.Expr) []ast.Expr {
	bin, ok := e.(*ast.BinaryExpr)
	if !ok || bin.Op != token.LOR {
		return []ast.Expr{e}
	}
	return append(orOperands(ast.Unparen(bin.X)), orOperands(ast.Unparen(bin.Y))...)
}

// nonNilErr returns the object of E when e is `E != nil` (or `nil != E`) and E is an error-typed identifier.
func nonNilErr(pass *analysis.Pass, e ast.Expr) types.Object {
	bin, ok := ast.Unparen(e).(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return nil
	}
	x, y := ast.Unparen(bin.X), ast.Unparen(bin.Y)
	if pass.TypesInfo.Types[x].IsNil() {
		x, y = y, x
	}
	id, ok := x.(*ast.Ident)
	if !ok || !pass.TypesInfo.Types[y].IsNil() {
		return nil
	}
	obj := pass.TypesInfo.ObjectOf(id)
	if obj == nil || !types.Implements(obj.Type(), errorType) {
		return nil
	}
	return obj
}

func checkBody(pass *analysis.Pass, body *ast.BlockStmt, obj types.Object) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.IfStmt:
			return nonNilErr(pass, n.Cond) != obj
		case *ast.CallExpr:
			if wrapsObj(pass, n, obj) {
				pass.Report(analysis.Diagnostic{
					Pos:     n.Pos(),
					Message: "%w wraps " + obj.Name() + ", but " + obj.Name() + " can be nil here because the if condition also passes when " + obj.Name() + " == nil; handle that case separately",
				})
			}
		}
		return true
	})
}

func wrapsObj(pass *analysis.Pass, call *ast.CallExpr, obj types.Object) bool {
	fn := typeutil.StaticCallee(pass.TypesInfo, call)
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != "fmt" || fn.Name() != "Errorf" || len(call.Args) < 2 {
		return false
	}
	v := pass.TypesInfo.Types[call.Args[0]].Value
	if v == nil || v.Kind() != constant.String {
		return false
	}
	verbs, args := wrapVerbs(constant.StringVal(v)), call.Args[1:]
	for i := range min(len(verbs), len(args)) {
		if id, ok := ast.Unparen(args[i]).(*ast.Ident); ok && verbs[i] && pass.TypesInfo.ObjectOf(id) == obj {
			return true
		}
	}
	return false
}

// wrapVerbs reports, per argument, whether its verb is %w. It returns nil for
// formats it cannot map simply (explicit indexes or '*').
func wrapVerbs(format string) []bool {
	var verbs []bool
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		for i++; i < len(format) && strings.IndexByte("+-# 0123456789.", format[i]) >= 0; i++ {
		}
		if i >= len(format) || format[i] == '%' {
			continue
		}
		if format[i] == '[' || format[i] == '*' {
			return nil
		}
		verbs = append(verbs, format[i] == 'w')
	}
	return verbs
}
