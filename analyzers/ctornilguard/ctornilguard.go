package ctornilguard

import (
	"fmt"
	"go/ast"
	"go/token"
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
		Name: "ctornilguard",
		Doc:  "checks constructor dependency guards and redundant method checks on constructor-validated dependencies (advisory, opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable advisory constructor nil guard checks")
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
	checkMethodGuards(pass)
	optional := nilCheckedFields(pass)

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			if !fn.Name.IsExported() || !strings.HasPrefix(fn.Name.Name, "New") {
				continue
			}
			if isDelegating(fn) {
				continue
			}

			var unguarded []string
			for _, field := range fn.Type.Params.List {
				t := pass.TypesInfo.TypeOf(field.Type)
				if t == nil || !isGuardableType(pass, t) {
					continue
				}
				for _, name := range field.Names {
					if name.Name == "_" {
						continue
					}
					if isUsed(fn, name.Name) && !hasNilCheck(fn, name.Name) &&
						!storedInOptionalField(pass, fn, pass.TypesInfo.ObjectOf(name), optional) {
						unguarded = append(unguarded, name.Name)
					}
				}
			}
			if len(unguarded) > 0 {
				pass.Report(analysis.Diagnostic{
					Pos: fn.Pos(),
					Message: fmt.Sprintf("%s stores dependencies without a nil guard: %s; add 'if p == nil' with an error return or a default",
						fn.Name.Name, strings.Join(unguarded, ", ")),
				})
			}
		}
	}
	return nil, nil
}

// nilCheckedFields returns struct fields that package code compares with nil.
// A dependency stored in such a field is optional: its users handle nil.
func nilCheckedFields(pass *analysis.Pass) map[*types.Var]bool {
	fields := make(map[*types.Var]bool)
	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			bin, ok := n.(*ast.BinaryExpr)
			if !ok || (bin.Op != token.EQL && bin.Op != token.NEQ) {
				return true
			}
			if sel, ok := comparedWithNil(pass, bin).(*ast.SelectorExpr); ok {
				if field, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Var); ok && field.IsField() {
					fields[field] = true
				}
			}
			return true
		})
	}
	return fields
}

// storedInOptionalField reports whether fn stores param in a field that
// package code nil-checks, through a keyed literal or a field assignment.
func storedInOptionalField(pass *analysis.Pass, fn *ast.FuncDecl, param types.Object, optional map[*types.Var]bool) bool {
	isParam := func(e ast.Expr) bool {
		id, ok := e.(*ast.Ident)
		return ok && pass.TypesInfo.ObjectOf(id) == param
	}
	isOptional := func(e ast.Expr) bool {
		var id *ast.Ident
		switch e := e.(type) {
		case *ast.Ident:
			id = e
		case *ast.SelectorExpr:
			id = e.Sel
		default:
			return false
		}
		field, ok := pass.TypesInfo.ObjectOf(id).(*types.Var)
		return ok && optional[field]
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.KeyValueExpr:
			found = found || (isParam(n.Value) && isOptional(n.Key))
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				if i < len(n.Rhs) && isParam(n.Rhs[i]) && isOptional(lhs) {
					found = true
				}
			}
		}
		return !found
	})
	return found
}

func isDelegating(fn *ast.FuncDecl) bool {
	if len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return false
	}
	for _, res := range ret.Results {
		if _, ok := res.(*ast.CallExpr); !ok {
			return false
		}
	}
	return true
}

func isGuardableType(pass *analysis.Pass, t types.Type) bool {
	if isContextType(t) {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return true
	}
	return false
}

func isContextType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

func isUsed(fn *ast.FuncDecl, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasNilCheck(fn *ast.FuncDecl, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		if bin.Op != token.EQL && bin.Op != token.NEQ {
			return true
		}
		left, lok := bin.X.(*ast.Ident)
		right, rok := bin.Y.(*ast.Ident)
		if lok && left.Name == name && rok && right.Name == "nil" {
			found = true
			return false
		}
		if rok && right.Name == name && lok && left.Name == "nil" {
			found = true
			return false
		}
		return true
	})
	return found
}
