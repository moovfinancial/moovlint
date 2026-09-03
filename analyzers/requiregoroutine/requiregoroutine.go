package requiregoroutine

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var fatalMethods = map[string]bool{
	"Fatal":    true,
	"Fatalf":   true,
	"FailNow":  true,
	"FailNowf": true,
	"Skip":     true,
	"Skipf":    true,
	"SkipNow":  true,
}

var Analyzer = &analysis.Analyzer{
	Name: "requiregoroutine",
	Doc:  "detects require.* and t.Fatal/FailNow calls inside goroutine closures (go statements, httptest handlers, callbacks) in test files",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if !moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		exempt := collectExemptFuncLits(file)

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.FuncLit)
			if !ok {
				return true
			}
			if exempt[lit.Pos()] {
				return true
			}
			if hasTestingTParam(pass, lit) {
				return true
			}
			checkClosureBody(pass, lit)
			return true
		})
	}
	return nil, nil
}

// collectExemptFuncLits returns function literals that run on the test
// goroutine: deferred closures and t.Cleanup callbacks.
func collectExemptFuncLits(file *ast.File) map[token.Pos]bool {
	exempt := make(map[token.Pos]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.DeferStmt:
			if lit, ok := node.Call.Fun.(*ast.FuncLit); ok {
				exempt[lit.Pos()] = true
			}
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Cleanup" {
				for _, arg := range node.Args {
					if lit, ok := arg.(*ast.FuncLit); ok {
						exempt[lit.Pos()] = true
					}
				}
			}
		}
		return true
	})
	return exempt
}

func hasTestingTParam(pass *analysis.Pass, lit *ast.FuncLit) bool {
	if lit.Type == nil || lit.Type.Params == nil || len(lit.Type.Params.List) == 0 {
		return false
	}
	t := pass.TypesInfo.TypeOf(lit.Type.Params.List[0].Type)
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	if obj.Pkg().Path() != "testing" {
		return false
	}
	switch obj.Name() {
	case "T", "B", "F":
		return true
	}
	return false
}

func checkClosureBody(pass *analysis.Pass, lit *ast.FuncLit) {
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		pkgPath := moovutil.SelectorPackagePath(pass, sel)
		if pkgPath == "github.com/stretchr/testify/require" && sel.Sel.Name != "New" {
			reportFailNow(pass, call)
			return true
		}
		if fatalMethods[sel.Sel.Name] && isTestingT(pass, sel.X) {
			reportFailNow(pass, call)
			return true
		}
		return true
	})
}

func isTestingT(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == "testing" && obj.Name() == "T"
}

func reportFailNow(pass *analysis.Pass, call *ast.CallExpr) {
	pass.Report(analysis.Diagnostic{
		Pos: call.Pos(),
		Message: "require/t.Fatal called from a non-test goroutine closure; FailNow cannot unwind the test goroutine, " +
			"return the error and assert on the test goroutine",
	})
}
