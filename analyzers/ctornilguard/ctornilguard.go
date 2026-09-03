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

var Analyzer = &analysis.Analyzer{
	Name: "ctornilguard",
	Doc:  "checks exported New* constructors nil-check pointer and interface dependencies before storing them",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

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
					if isUsed(fn, name.Name) && !hasNilCheck(fn, name.Name) {
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
