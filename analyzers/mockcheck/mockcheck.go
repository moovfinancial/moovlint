package mockcheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

type Config struct {
	AllowInterfaces string `json:"allow-interfaces"`
}

var Analyzer = New(Config{})

func New(cfg Config) *analysis.Analyzer {
	if cfg.AllowInterfaces == "" {
		cfg.AllowInterfaces = `Client$`
	}
	a := &analysis.Analyzer{
		Name: "mockcheck",
		Doc:  "detects test replacements for internal interfaces, including interfaces from other packages in the module",
	}
	a.Flags.StringVar(&cfg.AllowInterfaces, "allow-interfaces", cfg.AllowInterfaces, "regexp matching allowed interface names as import/path.Interface; use ^$ to allow none")
	a.Run = func(pass *analysis.Pass) (any, error) {
		allowed, err := regexp.Compile(cfg.AllowInterfaces)
		if err != nil {
			return nil, fmt.Errorf("allow-interfaces: %w", err)
		}
		return run(pass, allowed)
	}
	return a
}

func run(pass *analysis.Pass, allowed *regexp.Regexp) (any, error) {
	ifaces := collectInterfaces(pass)
	uses := interfaceUses(pass, allowed)

	for _, file := range pass.Files {
		if !strings.HasSuffix(pass.Fset.Position(file.Package).Filename, "_test.go") || ast.IsGenerated(file) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			_, ok = ts.Type.(*ast.StructType)
			if !ok {
				return true
			}

			obj := pass.TypesInfo.Defs[ts.Name]
			if obj == nil {
				return true
			}

			named, ok := obj.Type().(*types.Named)
			if !ok {
				return true
			}

			ptr := types.NewPointer(named)
			if iface := uses[named]; iface != nil {
				pass.Reportf(ts.Pos(), "test replacement '%s' implements internal interface '%s'; use the shared test environment with real services instead", ts.Name.Name, iface.Obj().Name())
				return true
			}
			if !isMockName(ts.Name.Name) {
				return true
			}
			for ifaceName, iface := range ifaces {
				if iface.NumMethods() == 0 || allowed.MatchString(pass.Pkg.Path()+"."+ifaceName) {
					continue
				}
				if types.Implements(ptr, iface) || types.Implements(named, iface) {
					pass.Report(analysis.Diagnostic{
						Pos:     ts.Pos(),
						Message: fmt.Sprintf("hand-rolled mock '%s' implements same-package interface '%s'; use test.NewEnvironment with real services or eventingtest instead", ts.Name.Name, ifaceName),
					})
					break
				}
			}

			return true
		})
	}
	return nil, nil
}

// Match actual dependency uses so incidental method overlap does not make a
// test helper a mock. Module metadata is required for cross-package matches.
func interfaceUses(pass *analysis.Pass, allowed *regexp.Regexp) map[*types.Named]*types.Named {
	uses := make(map[*types.Named]*types.Named)
	check := func(expr ast.Expr, target types.Type) {
		if target == nil {
			return
		}
		iface, ok := types.Unalias(target).(*types.Named)
		if !ok || iface.Obj().Pkg() == nil {
			return
		}
		it, ok := iface.Underlying().(*types.Interface)
		if !ok || it.NumMethods() == 0 {
			return
		}
		pkg := iface.Obj().Pkg().Path()
		if allowed.MatchString(pkg + "." + iface.Obj().Name()) {
			return
		}
		if pkg != pass.Pkg.Path() && (pass.Module == nil ||
			(pkg != pass.Module.Path && !strings.HasPrefix(pkg, pass.Module.Path+"/"))) {
			return
		}
		t := pass.TypesInfo.TypeOf(expr)
		if t == nil {
			return
		}
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		named, ok := types.Unalias(t).(*types.Named)
		if !ok || !declaresInterfaceMethod(named, it) {
			return
		}
		if _, ok := named.Underlying().(*types.Struct); ok {
			uses[named] = iface
		}
	}
	for _, file := range pass.Files {
		if !strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") || ast.IsGenerated(file) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if n.Body == nil || n.Type.Results == nil {
					break
				}
				sig := pass.TypesInfo.ObjectOf(n.Name).Type().(*types.Signature)
				ast.Inspect(n.Body, func(node ast.Node) bool {
					if _, ok := node.(*ast.FuncLit); ok {
						return false
					}
					if ret, ok := node.(*ast.ReturnStmt); ok && len(ret.Results) == sig.Results().Len() {
						for i, expr := range ret.Results {
							check(expr, sig.Results().At(i).Type())
						}
					}
					return true
				})
			case *ast.CallExpr:
				if pass.TypesInfo.Types[n.Fun].IsType() && len(n.Args) == 1 {
					check(n.Args[0], pass.TypesInfo.TypeOf(n.Fun))
					break
				}
				sig, ok := pass.TypesInfo.TypeOf(n.Fun).(*types.Signature)
				if !ok {
					break
				}
				for i, arg := range n.Args {
					index := i
					if sig.Variadic() && index >= sig.Params().Len()-1 {
						index = sig.Params().Len() - 1
					}
					if index >= sig.Params().Len() {
						break
					}
					t := sig.Params().At(index).Type()
					if sig.Variadic() && index == sig.Params().Len()-1 && !n.Ellipsis.IsValid() {
						t = t.(*types.Slice).Elem()
					}
					check(arg, t)
				}
			case *ast.ValueSpec:
				if n.Type != nil {
					for _, value := range n.Values {
						check(value, pass.TypesInfo.TypeOf(n.Type))
					}
				}
			case *ast.AssignStmt:
				if len(n.Lhs) == len(n.Rhs) {
					for i, value := range n.Rhs {
						check(value, pass.TypesInfo.TypeOf(n.Lhs[i]))
					}
				}
			case *ast.CompositeLit:
				st, ok := pass.TypesInfo.TypeOf(n).Underlying().(*types.Struct)
				if !ok {
					break
				}
				for i, elt := range n.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						key, ok := kv.Key.(*ast.Ident)
						if ok {
							if field := pass.TypesInfo.ObjectOf(key); field != nil {
								check(kv.Value, field.Type())
							}
						}
					} else if i < st.NumFields() {
						check(elt, st.Field(i).Type())
					}
				}
			}
			return true
		})
	}
	return uses
}

func declaresInterfaceMethod(named *types.Named, iface *types.Interface) bool {
	for i := 0; i < named.NumMethods(); i++ {
		for j := 0; j < iface.NumMethods(); j++ {
			if named.Method(i).Id() == iface.Method(j).Id() {
				return true
			}
		}
	}
	return false
}

func isMockName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "mock") || strings.HasPrefix(lower, "fake") || strings.HasPrefix(lower, "stub")
}

func collectInterfaces(pass *analysis.Pass) map[string]*types.Interface {
	ifaces := make(map[string]*types.Interface)
	scope := pass.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			continue
		}
		iface, ok := named.Underlying().(*types.Interface)
		if !ok {
			continue
		}
		ifaces[name] = iface
	}
	return ifaces
}
