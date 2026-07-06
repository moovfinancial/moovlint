package mockcheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "mockcheck",
	Doc:  "detects hand-rolled mock/fake/stub types in test files that implement same-package interfaces",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	ifaces := collectInterfaces(pass)

	for _, file := range pass.Files {
		if !strings.HasSuffix(pass.Fset.Position(file.Package).Filename, "_test.go") {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			if !isMockName(ts.Name.Name) {
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
			for ifaceName, iface := range ifaces {
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
