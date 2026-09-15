package spanerrors

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "spanerrors",
	Doc:  "checks that functions which create a span record returned errors with telemetry.RecordError before returning",
	Run:  run,
}

var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

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
			if !ok || fn.Body == nil || fn.Type.Results == nil {
				continue
			}
			if !signatureReturnsError(pass, fn) {
				continue
			}
			if !hasNonNilErrorReturn(pass, fn) {
				continue
			}
			if !createsSpan(pass, fn) {
				continue
			}
			if recordsError(pass, fn) {
				continue
			}
			pass.Report(analysis.Diagnostic{
				Pos: fn.Pos(),
				Message: fmt.Sprintf("%s creates a span but returns errors without recording them; call telemetry.RecordError " +
					"(or RecordErrorAtLow for expected errors) before returning", fn.Name.Name),
			})
		}
	}
	return nil, nil
}

func signatureReturnsError(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	for _, result := range fn.Type.Results.List {
		t := pass.TypesInfo.TypeOf(result.Type)
		if t != nil && types.Implements(t, errorIface) {
			return true
		}
	}
	return false
}

func hasNonNilErrorReturn(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, res := range ret.Results {
			if isNil(res) {
				continue
			}
			t := pass.TypesInfo.TypeOf(res)
			if t != nil && types.Implements(t, errorIface) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func createsSpan(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
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
		if moovutil.IsTelemetryPackage(moovutil.SelectorPackagePath(pass, sel)) {
			found = true
			return false
		}
		return true
	})
	return found
}

func recordsError(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		pkgPath := moovutil.SelectorPackagePath(pass, sel)
		if moovutil.IsTelemetryPackage(pkgPath) && strings.HasPrefix(name, "RecordError") {
			found = true
			return false
		}
		if (name == "RecordError" || name == "SetStatus") &&
			(pkgPath == "go.opentelemetry.io/otel/trace" || strings.HasSuffix(pkgPath, "/otel/trace")) {
			found = true
			return false
		}
		return true
	})
	return found
}

func isNil(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "nil"
}
