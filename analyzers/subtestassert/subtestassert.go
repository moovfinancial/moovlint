package subtestassert

import (
	"go/ast"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "subtestassert",
	Doc:  "detects assertion objects created from the outer test's t used inside t.Run closures; failures bypass the subtest",
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
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Run" {
				return true
			}
			for _, arg := range call.Args {
				lit, ok := arg.(*ast.FuncLit)
				if !ok {
					continue
				}
				if !hasTestingTParam(pass, lit) {
					continue
				}
				checkSubtestBody(pass, lit)
			}
			return true
		})
	}
	return nil, nil
}

func checkSubtestBody(pass *analysis.Pass, lit *ast.FuncLit) {
	localDefs := collectLocalDefs(lit)

	ast.Inspect(lit.Body, func(n ast.Node) bool {
		if inner, ok := n.(*ast.FuncLit); ok && hasTestingTParam(pass, inner) {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !isTestifyAssertionsType(pass.TypesInfo.TypeOf(sel.X)) {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && localDefs[id.Name] {
			return true
		}
		pass.Report(analysis.Diagnostic{
			Pos: call.Pos(),
			Message: "assertion object from the outer test is used inside t.Run; create assertions from the subtest's t " +
				"so failures stop the right test",
		})
		return true
	})
}

func collectLocalDefs(lit *ast.FuncLit) map[string]bool {
	defs := make(map[string]bool)
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					defs[id.Name] = true
				}
			}
		case *ast.DeclStmt:
			gd, ok := node.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					defs[name.Name] = true
				}
			}
		}
		return true
	})
	return defs
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
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == "testing" && obj.Name() == "T"
}

func isTestifyAssertionsType(t types.Type) bool {
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
	if obj.Name() != "Assertions" {
		return false
	}
	p := obj.Pkg().Path()
	return p == "github.com/stretchr/testify/require" || p == "github.com/stretchr/testify/assert"
}
