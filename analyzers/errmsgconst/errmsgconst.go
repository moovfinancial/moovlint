package errmsgconst

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

type Config struct {
	Enabled bool `json:"enabled"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "errmsgconst",
		Doc:  "detects error messages built from non-constant values; keep the message constant and record values as span attributes (opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable constant error message checks")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		return run(pass)
	}
	return a
}

var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

const message = "error message includes %s; keep the message constant and record the value as a span attribute " +
	"so GROUP BY exception.message groups the same failure"

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
			switch {
			case isFunc(pass, call, "fmt", "Errorf"):
				if arg := dataArg(pass, call); arg != nil {
					pass.Reportf(arg.Pos(), message, types.ExprString(arg))
				}
			case isFunc(pass, call, "errors", "New") && len(call.Args) == 1:
				inner, ok := ast.Unparen(call.Args[0]).(*ast.CallExpr)
				if ok && isFunc(pass, inner, "fmt", "Sprintf") {
					if arg := dataArg(pass, inner); arg != nil {
						pass.Reportf(arg.Pos(), message, types.ExprString(arg))
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

// dataArg returns the first argument formatted by a verb other than %w that
// is neither an error nor a constant. Errors are allowed with any verb: %v on
// an error is the usual way to drop its flags.
func dataArg(pass *analysis.Pass, call *ast.CallExpr) ast.Expr {
	if len(call.Args) < 2 {
		return nil
	}
	tv := pass.TypesInfo.Types[call.Args[0]]
	if tv.Value == nil || tv.Value.Kind() != constant.String {
		return nil
	}
	verbs := formatVerbs(constant.StringVal(tv.Value))
	args := call.Args[1:]
	if verbs == nil || len(verbs) != len(args) {
		return nil
	}
	for i, arg := range args {
		if verbs[i] == 'w' || pass.TypesInfo.Types[arg].Value != nil {
			continue
		}
		if t := pass.TypesInfo.TypeOf(arg); t != nil && types.Implements(t, errorIface) {
			continue
		}
		return arg
	}
	return nil
}

// formatVerbs returns the verb letters in order, or nil when explicit
// argument indexes or * widths make the mapping unclear.
func formatVerbs(format string) []byte {
	verbs := []byte{}
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
		verbs = append(verbs, format[i])
	}
	return verbs
}

func isFunc(pass *analysis.Pass, call *ast.CallExpr, pkg, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	path := moovutil.SelectorPackagePath(pass, sel)
	if pkg == "errors" {
		return path == "errors" || moovutil.IsErrorsPackage(path)
	}
	return path == pkg
}
