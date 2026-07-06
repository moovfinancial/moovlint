package spanrequired

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "spanrequired",
	Doc:  "checks that exported methods on service/repo structs starting with a context parameter also start a span",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Package).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || !funcDecl.Name.IsExported() {
				return true
			}

			if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				return true
			}

			recvType := funcDecl.Recv.List[0].Type
			recvObj := typeToObject(pass, recvType)
			if recvObj == nil || recvObj.Pkg() == nil {
				return true
			}

			recvPkg := recvObj.Pkg().Path()
			if !isServicePackage(recvPkg) {
				return true
			}

			if !hasContextFirstParam(funcDecl) {
				return true
			}

			if hasSpanCall(funcDecl) {
				return true
			}

			pass.Report(analysis.Diagnostic{
				Pos:     funcDecl.Pos(),
				Message: fmt.Sprintf("exported method %s.%s takes context but does not start a telemetry span; add ctx, span := telemetry.StartSpan(ctx, \"%s\")",
					recvObj.Name(), funcDecl.Name.Name, lowerKebab(funcDecl.Name.Name)),
			})
			return true
		})
	}
	return nil, nil
}

func typeToObject(pass *analysis.Pass, expr ast.Expr) *types.TypeName {
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok {
		return nil
	}
	t := tv.Type
	for {
		ptr, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	return named.Obj()
}

func hasContextFirstParam(fn *ast.FuncDecl) bool {
	params := fn.Type.Params
	if params == nil || len(params.List) == 0 {
		return false
	}
	first := params.List[0]
	sel, ok := first.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkgIdent.Name == "context" && sel.Sel.Name == "Context"
}

func hasSpanCall(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
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
		name := sel.Sel.Name
		if name == "StartSpan" || name == "StartLinkedRootSpan" {
			found = true
			return false
		}
		return true
	})
	return found
}

func isServicePackage(pkgPath string) bool {
	// Service/repo/client/consumer packages live under a moovfinancial or moov-io root.
	// We exclude common non-service patterns.
	if strings.Contains(pkgPath, "test") && strings.HasSuffix(pkgPath, "test") {
		return false
	}
	excluded := []string{
		"/cmd/",
		"/docs/",
		"/examples/",
		"/scripts/",
		"/mocks",
		"/mock",
	}
	for _, e := range excluded {
		if strings.Contains(pkgPath, e) {
			return false
		}
	}
	return true
}

func lowerKebab(name string) string {
	var result []rune
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(strings.ReplaceAll(string(result), "_", "-"))
}
