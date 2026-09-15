package ctornilguard

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

// Limit this check to private implementations with direct constructor returns.
// A second construction path or a field write cancels the constructor evidence.
func checkMethodGuards(pass *analysis.Pass) {
	guarded := make(map[*ast.CompositeLit]map[*types.Var]bool)
	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Pos()).Filename) || ast.IsGenerated(file) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "New") {
				continue
			}
			collectGuardedLiterals(pass, fn, guarded)
		}
	}
	required := make(map[*types.Var]bool)
	invalid := make(map[*types.Var]bool)
	for _, fields := range guarded {
		for field := range fields {
			required[field] = true
		}
	}
	invalidateType := func(t types.Type) {
		if t == nil {
			return
		}
		if st, ok := t.Underlying().(*types.Struct); ok {
			for i := 0; i < st.NumFields(); i++ {
				invalid[st.Field(i)] = true
			}
		}
	}
	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Pos()).Filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CompositeLit:
				st, ok := pass.TypesInfo.TypeOf(n).Underlying().(*types.Struct)
				if ok {
					for i := 0; i < st.NumFields(); i++ {
						field := st.Field(i)
						if !guarded[n][field] {
							invalid[field] = true
						}
					}
				}
			case *ast.AssignStmt:
				for _, lhs := range n.Lhs {
					invalidateType(pass.TypesInfo.TypeOf(lhs))
					if sel, ok := lhs.(*ast.SelectorExpr); ok {
						if field, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Var); ok {
							invalid[field] = true
						}
					}
				}
			case *ast.ValueSpec:
				for _, name := range n.Names {
					invalidateType(pass.TypesInfo.TypeOf(name))
				}
			case *ast.UnaryExpr:
				if n.Op == token.AND {
					if sel, ok := n.X.(*ast.SelectorExpr); ok {
						if field, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Var); ok {
							invalid[field] = true
						}
					}
				}
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok {
					if obj, ok := pass.TypesInfo.ObjectOf(id).(*types.Builtin); ok && obj.Name() == "new" && len(n.Args) == 1 {
						invalidateType(pass.TypesInfo.TypeOf(n.Args[0]))
					}
				}
			}
			return true
		})
	}
	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Pos()).Filename) || ast.IsGenerated(file) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv == nil || len(fn.Recv.List[0].Names) != 1 {
				continue
			}
			receiver := pass.TypesInfo.ObjectOf(fn.Recv.List[0].Names[0])
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				bin, ok := n.(*ast.BinaryExpr)
				if !ok || (bin.Op != token.EQL && bin.Op != token.NEQ) {
					return true
				}
				expr := comparedWithNil(pass, bin)
				sel, ok := expr.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				id, ok := sel.X.(*ast.Ident)
				if !ok || pass.TypesInfo.ObjectOf(id) != receiver {
					return true
				}
				field, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Var)
				if ok && required[field] && !invalid[field] {
					pass.Reportf(bin.Pos(), "dependency %s is already nil-checked by its constructor; keep required dependency checks in the constructor", field.Name())
				}
				return true
			})
		}
	}
}

func collectGuardedLiterals(pass *analysis.Pass, fn *ast.FuncDecl, literals map[*ast.CompositeLit]map[*types.Var]bool) {
	params := make(map[types.Object]bool)
	for _, param := range fn.Type.Params.List {
		for _, name := range param.Names {
			params[pass.TypesInfo.ObjectOf(name)] = true
		}
	}
	checked := make(map[types.Object]bool)
	for _, stmt := range fn.Body.List {
		switch stmt := stmt.(type) {
		case *ast.IfStmt:
			bin, ok := stmt.Cond.(*ast.BinaryExpr)
			if !ok || bin.Op != token.EQL || stmt.Init != nil || stmt.Else != nil || !returnsNilError(pass, stmt.Body) {
				return
			}
			id, ok := comparedWithNil(pass, bin).(*ast.Ident)
			if !ok || !params[pass.TypesInfo.ObjectOf(id)] {
				return
			}
			checked[pass.TypesInfo.ObjectOf(id)] = true
		case *ast.ReturnStmt:
			if len(stmt.Results) == 0 {
				return
			}
			expr := stmt.Results[0]
			if ptr, ok := expr.(*ast.UnaryExpr); ok && ptr.Op == token.AND {
				expr = ptr.X
			}
			lit, ok := expr.(*ast.CompositeLit)
			if !ok {
				return
			}
			named, ok := pass.TypesInfo.TypeOf(lit).(*types.Named)
			if !ok || named.Obj().Exported() {
				return
			}
			fields := make(map[*types.Var]bool)
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				id, ok := kv.Value.(*ast.Ident)
				if !ok || !checked[pass.TypesInfo.ObjectOf(id)] {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				field, ok := pass.TypesInfo.ObjectOf(key).(*types.Var)
				if ok && !field.Exported() {
					fields[field] = true
				}
			}
			literals[lit] = fields
		default:
			// More complex construction needs control-flow analysis before we
			// can treat a dependency as required on every return path.
			return
		}
	}
}

func comparedWithNil(pass *analysis.Pass, bin *ast.BinaryExpr) ast.Expr {
	if pass.TypesInfo.Types[bin.X].IsNil() {
		return bin.Y
	}
	if pass.TypesInfo.Types[bin.Y].IsNil() {
		return bin.X
	}
	return nil
}

func returnsNilError(pass *analysis.Pass, body *ast.BlockStmt) bool {
	if len(body.List) != 1 {
		return false
	}
	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 || !pass.TypesInfo.Types[ret.Results[0]].IsNil() {
		return false
	}
	call, ok := ret.Results[1].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg := moovutil.SelectorPackagePath(pass, sel)
	return (sel.Sel.Name == "New" && (pkg == "errors" || pkg == "github.com/moovfinancial/errors")) ||
		(sel.Sel.Name == "Errorf" && pkg == "fmt")
}
