package spancontext

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "spancontext",
	Doc:  "detects End() or SetName() calls on spans retrieved from context via trace.SpanFromContext",
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

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name != "End" && sel.Sel.Name != "SetName" {
				return true
			}

			if isSpanFromContextCall(pass, sel.X) {
				pass.Report(analysis.Diagnostic{
					Pos:     call.Pos(),
					Message: fmt.Sprintf("do not call %s on a span retrieved from context; spans from context are owned by their creator", sel.Sel.Name),
				})
				return true
			}

			if id, ok := sel.X.(*ast.Ident); ok {
				if isSpanFromContextVar(pass, file, id.Name) {
					pass.Report(analysis.Diagnostic{
						Pos:     call.Pos(),
						Message: fmt.Sprintf("do not call %s on '%s' which is a span retrieved from context; spans from context are owned by their creator", sel.Sel.Name, id.Name),
					})
				}
			}
			return true
		})
	}
	return nil, nil
}

func isSpanFromContextCall(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "SpanFromContext" {
		return false
	}
	pkgPath := moovutil.SelectorPackagePath(pass, sel)
	return isTracePackage(pkgPath)
}

func isSpanFromContextVar(pass *analysis.Pass, file *ast.File, varName string) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			if i >= len(assign.Lhs) {
				break
			}
			id, ok := assign.Lhs[i].(*ast.Ident)
			if !ok || id.Name != varName {
				continue
			}
			if isSpanFromContextCall(pass, rhs) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func isTracePackage(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, "/otel/trace") ||
		pkgPath == "go.opentelemetry.io/otel/trace"
}
