package spannertxcapture

import (
	"go/ast"
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
		Name: "spannertxcapture",
		Doc:  "detects Spanner read-write transaction closures that write captured variables without a reset, so retries see stale state (opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable Spanner transaction capture checks")
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
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "ReadWriteTransaction" && sel.Sel.Name != "ReadWriteTransactionWithOptions") {
				return true
			}
			for _, arg := range call.Args {
				if lit, ok := arg.(*ast.FuncLit); ok && hasTxParam(pass, lit) {
					(&checker{pass: pass, lit: lit, reset: map[*types.Var]bool{}, reported: map[*types.Var]bool{}}).check()
				}
			}
			return true
		})
	}
	return nil, nil
}

var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func hasTxParam(pass *analysis.Pass, lit *ast.FuncLit) bool {
	for _, field := range lit.Type.Params.List {
		ptr, ok := pass.TypesInfo.TypeOf(field.Type).(*types.Pointer)
		if !ok {
			continue
		}
		named, ok := ptr.Elem().(*types.Named)
		if ok && named.Obj().Name() == "ReadWriteTransaction" && named.Obj().Pkg() != nil &&
			named.Obj().Pkg().Path() == "cloud.google.com/go/spanner" {
			return true
		}
	}
	return false
}

type checker struct {
	pass     *analysis.Pass
	lit      *ast.FuncLit
	reset    map[*types.Var]bool
	reported map[*types.Var]bool
}

func (c *checker) check() {
	for _, stmt := range c.lit.Body.List {
		c.walk(stmt, false)
		if as, ok := stmt.(*ast.AssignStmt); ok && as.Tok == token.ASSIGN && len(as.Lhs) == len(as.Rhs) {
			for i, lhs := range as.Lhs {
				if v := c.outerVar(lhs); v != nil && isReset(c.pass, v, as.Rhs[i]) {
					c.reset[v] = true
				}
			}
		}
	}
}

// walk reports risky writes in n. cond is true when n is nested in control flow.
func (c *checker) walk(n ast.Node, cond bool) {
	ast.Inspect(n, func(m ast.Node) bool {
		switch m := m.(type) {
		case *ast.FuncLit:
			return false
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.ForStmt, *ast.RangeStmt:
			if !cond {
				c.walk(m, true)
				return false
			}
		case *ast.IncDecStmt:
			c.write(m.X, true)
		case *ast.AssignStmt:
			if m.Tok == token.DEFINE {
				return true
			}
			for i, lhs := range m.Lhs {
				acc := m.Tok != token.ASSIGN
				if _, ok := lhs.(*ast.IndexExpr); ok {
					acc = true
				}
				if v := c.outerVar(lhs); v != nil && len(m.Lhs) == len(m.Rhs) && isSelfAppend(c.pass, v, m.Rhs[i]) {
					acc = true
				}
				c.write(lhs, acc || cond)
			}
		}
		return true
	})
}

func (c *checker) write(lhs ast.Expr, risky bool) {
	v := c.outerVar(lhs)
	if !risky || v == nil || c.reset[v] || c.reported[v] {
		return
	}
	c.reported[v] = true
	c.pass.Reportf(lhs.Pos(), "%s is written inside the Spanner transaction closure without a reset at the start of the closure; a retry reruns the closure with the value from the failed attempt", v.Name())
}

// outerVar returns the variable written by lhs (x or x[k]) if it is declared outside the closure.
func (c *checker) outerVar(lhs ast.Expr) *types.Var {
	if idx, ok := lhs.(*ast.IndexExpr); ok {
		lhs = idx.X
	}
	id, ok := ast.Unparen(lhs).(*ast.Ident)
	if !ok {
		return nil
	}
	v, ok := c.pass.TypesInfo.ObjectOf(id).(*types.Var)
	if !ok || (v.Pos() >= c.lit.Pos() && v.Pos() < c.lit.End()) {
		return nil
	}
	// A captured err is normally overwritten by the transaction's own return.
	if types.Implements(v.Type(), errorIface) {
		return nil
	}
	return v
}

func isSelfAppend(pass *analysis.Pass, v *types.Var, rhs ast.Expr) bool {
	call, ok := ast.Unparen(rhs).(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return false
	}
	fn, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok {
		return false
	}
	if _, ok := pass.TypesInfo.Uses[fn].(*types.Builtin); !ok || fn.Name != "append" {
		return false
	}
	return refers(pass, v, call.Args[0])
}

// isReset reports whether rhs gives v a fresh value: it does not refer to v, or it is v[:0].
func isReset(pass *analysis.Pass, v *types.Var, rhs ast.Expr) bool {
	if s, ok := ast.Unparen(rhs).(*ast.SliceExpr); ok && refers(pass, v, s.X) {
		if lit, ok := s.High.(*ast.BasicLit); ok && lit.Value == "0" && s.Low == nil {
			return true
		}
	}
	return !refers(pass, v, rhs)
}

func refers(pass *analysis.Pass, v *types.Var, n ast.Node) bool {
	found := false
	ast.Inspect(n, func(m ast.Node) bool {
		if id, ok := m.(*ast.Ident); ok && pass.TypesInfo.Uses[id] == v {
			found = true
		}
		return !found
	})
	return found
}
