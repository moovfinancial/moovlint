package subtestassert

import (
	"go/ast"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

// Config holds opt-in extensions. The base check always runs.
type Config struct {
	// OuterT also flags uses of an outer *testing.T inside a t.Run closure,
	// directly or through a captured struct field.
	OuterT bool `json:"outert"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "subtestassert",
		Doc:  "detects assertion objects created from the outer test's t used inside t.Run closures; failures bypass the subtest",
	}
	a.Flags.BoolVar(&cfg.OuterT, "outert", cfg.OuterT, "also flag the outer *testing.T used inside t.Run closures")
	a.Run = func(pass *analysis.Pass) (any, error) { return run(pass, cfg) }
	return a
}

func run(pass *analysis.Pass, cfg Config) (any, error) {
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
				if cfg.OuterT {
					checkOuterT(pass, lit)
				}
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

// checkOuterT flags *testing.T values declared outside the subtest closure
// and used inside it: the outer t itself, or a field such as scope.T.
func checkOuterT(pass *analysis.Pass, lit *ast.FuncLit) {
	outside := func(id *ast.Ident) bool {
		obj := pass.TypesInfo.ObjectOf(id)
		return obj != nil && obj.Parent() != types.Universe && (obj.Pos() < lit.Pos() || obj.Pos() >= lit.End())
	}
	ast.Inspect(lit.Body, func(n ast.Node) bool {
		if inner, ok := n.(*ast.FuncLit); ok && hasTestingTParam(pass, inner) {
			return false
		}
		switch e := n.(type) {
		case *ast.SelectorExpr:
			if !isTestingT(pass.TypesInfo.TypeOf(e)) {
				return true
			}
			if root, ok := ast.Unparen(e.X).(*ast.Ident); ok && outside(root) {
				reportOuterT(pass, e)
			}
			return false
		case *ast.Ident:
			if v, ok := pass.TypesInfo.ObjectOf(e).(*types.Var); ok && !v.IsField() && isTestingT(v.Type()) && outside(e) {
				reportOuterT(pass, e)
			}
		}
		return true
	})
}

func reportOuterT(pass *analysis.Pass, n ast.Node) {
	pass.Report(analysis.Diagnostic{
		Pos:     n.Pos(),
		Message: "the outer test's t is used inside t.Run; use the subtest's t so failures stop the right test",
	})
}

func isTestingT(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "testing" && named.Obj().Name() == "T"
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
