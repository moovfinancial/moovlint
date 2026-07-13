package controllerassert

import (
	"go/ast"
	"go/token"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "controllerassert",
	Doc:  "checks that HTTP controller structs with AppendRoutes have a compile-time interface assertion",
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

		controllers := findControllerStructs(file)
		if len(controllers) == 0 {
			continue
		}

		assertedTypes := findAssertedTypeNames(file)
		for _, c := range controllers {
			if assertedTypes[c.name] {
				continue
			}
			pass.Report(analysis.Diagnostic{
				Pos:     c.spec.Pos(),
				Message: "controller " + c.name + " has AppendRoutes but no compile-time interface assertion (var _ Interface = &" + c.name + "{}) was found in this file",
			})
		}
	}
	return nil, nil
}

type controllerInfo struct {
	name string
	spec *ast.TypeSpec
}

func findControllerStructs(file *ast.File) []controllerInfo {
	var controllers []controllerInfo
	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		_, ok = typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		if hasAppendRoutesMethod(file, typeSpec.Name.Name) {
			controllers = append(controllers, controllerInfo{name: typeSpec.Name.Name, spec: typeSpec})
		}
		return true
	})
	return controllers
}

func hasAppendRoutesMethod(file *ast.File, structName string) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}
		recvType := fn.Recv.List[0].Type
		if star, ok := recvType.(*ast.StarExpr); ok {
			recvType = star.X
		}
		ident, ok := recvType.(*ast.Ident)
		if !ok || ident.Name != structName {
			return true
		}
		if fn.Name != nil && fn.Name.Name == "AppendRoutes" {
			found = true
			return false
		}
		return true
	})
	return found
}

func findAssertedTypeNames(file *ast.File) map[string]bool {
	result := make(map[string]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || valueSpec.Values == nil {
				continue
			}
			for _, val := range valueSpec.Values {
				if name := extractAssertedTypeName(val); name != "" {
					result[name] = true
				}
			}
		}
		return true
	})
	return result
}

func extractAssertedTypeName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.UnaryExpr:
		if v.Op == token.AND {
			return extractTypeName(v.X)
		}
	case *ast.CallExpr:
		if len(v.Args) == 1 {
			if id, ok := v.Args[0].(*ast.Ident); ok && id.Name == "nil" {
				return extractTypeName(v.Fun)
			}
		}
	case *ast.CompositeLit:
		return extractTypeName(v.Type)
	}
	return ""
}

func extractTypeName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return extractTypeName(v.X)
	case *ast.CompositeLit:
		return extractTypeName(v.Type)
	case *ast.ParenExpr:
		return extractTypeName(v.X)
	}
	return ""
}
