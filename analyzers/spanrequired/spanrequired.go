package spanrequired

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
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
		if moovutil.IsTestFile(filename) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || !funcDecl.Name.IsExported() {
				return true
			}

			recvObj := moovutil.ReceiverTypeName(pass, funcDecl)
			if recvObj == nil || recvObj.Pkg() == nil {
				return true
			}

			if !moovutil.IsServicePackage(recvObj.Pkg().Path()) {
				return true
			}

			if !moovutil.HasContextFirstParam(pass, funcDecl) {
				return true
			}

			if hasTelemetrySpanCall(pass, funcDecl) {
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

func hasTelemetrySpanCall(pass *analysis.Pass, fn *ast.FuncDecl) bool {
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
		if name != "StartSpan" && name != "StartLinkedRootSpan" {
			return true
		}
		pkgPath := moovutil.SelectorPackagePath(pass, sel)
		if moovutil.IsTelemetryPackage(pkgPath) {
			found = true
			return false
		}
		return true
	})
	return found
}

func lowerKebab(name string) string {
	var result []rune
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '-')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
