package enumliteral

import (
	"go/ast"
	"go/constant"
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
		Name: "enumliteral",
		Doc:  "detects raw string literals used where a typed enum constant exists for that value (opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable enum literal checks")
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

	enums := map[*types.TypeName]map[string]string{}
	reported := map[*ast.BasicLit]bool{}

	check := func(lit *ast.BasicLit, t types.Type) {
		if reported[lit] {
			return
		}
		named, ok := t.(*types.Named)
		if !ok {
			return
		}
		consts := enumConsts(enums, named)
		name, ok := consts[constant.StringVal(constant.MakeFromLiteral(lit.Value, token.STRING, 0))]
		if !ok {
			return
		}
		if pkg := named.Obj().Pkg(); pkg != pass.Pkg {
			name = pkg.Name() + "." + name
		}
		reported[lit] = true
		pass.Reportf(lit.Pos(), "use %s instead of the string literal %s", name, lit.Value)
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.GenDecl:
				return n.Tok != token.CONST
			case *ast.CallExpr:
				// Explicit conversion T("value").
				if len(n.Args) != 1 {
					return true
				}
				lit, ok := ast.Unparen(n.Args[0]).(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				if tv, ok := pass.TypesInfo.Types[n.Fun]; ok && tv.IsType() {
					check(lit, tv.Type)
				}
			case *ast.BasicLit:
				// Implicit conversion: comparison, case clause, assignment, argument, return.
				if n.Kind == token.STRING {
					check(n, pass.TypesInfo.TypeOf(n))
				}
			}
			return true
		})
	}
	return nil, nil
}

// enumConsts returns the value->constant-name map for t, or nil if t is not
// a Moov string enum type.
func enumConsts(cache map[*types.TypeName]map[string]string, t *types.Named) map[string]string {
	obj := t.Obj()
	if consts, ok := cache[obj]; ok {
		return consts
	}
	var consts map[string]string
	basic, ok := t.Underlying().(*types.Basic)
	if ok && basic.Info()&types.IsString != 0 && obj.Pkg() != nil && moovutil.IsMoovPackage(obj.Pkg().Path()) {
		scope := obj.Pkg().Scope()
		for _, n := range scope.Names() {
			c, ok := scope.Lookup(n).(*types.Const)
			if !ok || !types.Identical(c.Type(), t) || c.Val().Kind() != constant.String {
				continue
			}
			if consts == nil {
				consts = map[string]string{}
			}
			if _, dup := consts[constant.StringVal(c.Val())]; !dup {
				consts[constant.StringVal(c.Val())] = c.Name()
			}
		}
	}
	cache[obj] = consts
	return consts
}
