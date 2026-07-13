package moovutil

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// IsMoovPackage returns true if the package path is under a Moov organization root
// or under the testdata module (for analyzer self-tests).
func IsMoovPackage(pkgPath string) bool {
	return strings.HasPrefix(pkgPath, "github.com/moovfinancial/") ||
		strings.HasPrefix(pkgPath, "github.com/moov-io/") ||
		strings.HasPrefix(pkgPath, "testdata/")
}

// IsServicePackage returns true if the package path is a Moov service package
// suitable for service/repo-level lint rules. It excludes non-service directories.
func IsServicePackage(pkgPath string) bool {
	if !IsMoovPackage(pkgPath) {
		return false
	}
	excluded := []string{
		"/cmd/",
		"/docs/",
		"/examples/",
		"/scripts/",
		"/mocks",
		"/mock",
		"/client",
		"/admin",
		"/test/",
	}
	for _, e := range excluded {
		if strings.Contains(pkgPath, e) {
			return false
		}
	}
	if strings.HasSuffix(pkgPath, "test") || strings.HasSuffix(pkgPath, "_test") {
		return false
	}
	return true
}

// IsTestFile returns true if the filename ends with _test.go.
func IsTestFile(filename string) bool {
	return strings.HasSuffix(filename, "_test.go")
}

// IsTelemetryPackage returns true if the package path is a Moov telemetry package.
func IsTelemetryPackage(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, "/observability/telemetry") ||
		pkgPath == "github.com/moov-io/base/telemetry" ||
		pkgPath == "github.com/moovfinancial/go-libs/observability/telemetry"
}

// IsMoovLogPackage returns true if the package path is a Moov log package.
func IsMoovLogPackage(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, "/observability/log") ||
		pkgPath == "github.com/moov-io/base/log" ||
		pkgPath == "github.com/moovfinancial/go-libs/observability/log"
}

// IsErrorsPackage returns true if the package path is the Moov errors package.
func IsErrorsPackage(pkgPath string) bool {
	return pkgPath == "github.com/moovfinancial/errors"
}

// IsMidPackage returns true if the package path is the Moov mid package.
func IsMidPackage(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, "/go-libs/mid") ||
		pkgPath == "github.com/moovfinancial/mid"
}

// IsMValidationPackage returns true if the package path is the Moov mvalidation package.
func IsMValidationPackage(pkgPath string) bool {
	return strings.HasSuffix(pkgPath, "/go-libs/mvalidation") ||
		pkgPath == "github.com/moovfinancial/mvalidation"
}

// SelectorPackagePath resolves the import path of the package that defines
// the selected symbol. Returns "" if it cannot be determined or if the
// selector is a field access (not a package-qualified identifier).
func SelectorPackagePath(pass *analysis.Pass, sel *ast.SelectorExpr) string {
	if sel.Sel == nil {
		return ""
	}
	obj := pass.TypesInfo.ObjectOf(sel.Sel)
	if obj == nil {
		return ""
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return ""
	}
	return pkg.Path()
}

// ReceiverTypeName resolves the named type of a function declaration's receiver,
// stripping pointer wrappers. Returns nil if the receiver is not a named type.
func ReceiverTypeName(pass *analysis.Pass, fn *ast.FuncDecl) *types.TypeName {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return nil
	}
	return ExprToTypeName(pass, fn.Recv.List[0].Type)
}

// ExprToTypeName resolves an ast.Expr to a *types.TypeName, stripping pointers.
func ExprToTypeName(pass *analysis.Pass, expr ast.Expr) *types.TypeName {
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

// HasContextFirstParam returns true if the function's first parameter is
// context.Context (matched by type, not just by name).
func HasContextFirstParam(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	params := fn.Type.Params
	if params == nil || len(params.List) == 0 {
		return false
	}
	first := params.List[0]
	t := pass.TypesInfo.TypeOf(first.Type)
	if t == nil {
		return false
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	return named.Obj().Pkg() != nil &&
		named.Obj().Pkg().Path() == "context" &&
		named.Obj().Name() == "Context"
}
