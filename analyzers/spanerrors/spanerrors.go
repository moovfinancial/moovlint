package spanerrors

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

type Config struct {
	Enabled bool `json:"enabled"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "spanerrors",
		Doc:  "checks that an error created in a function which creates a span is recorded there with telemetry.RecordError; errors passed up from callees are not flagged (advisory, opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable advisory span error recording checks")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		return run(pass)
	}
	return a
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
			if spanPos := spanStart(pass, fn); spanPos.IsValid() {
				checkReturns(pass, fn, spanPos)
			}
		}
	}
	return nil, nil
}

// record is a RecordError call and the innermost block that contains it.
type record struct {
	pos   token.Pos
	scope ast.Node
}

// checkReturns reports each return after the span starts that returns a
// non-nil error with no RecordError on its path. A return is covered when the
// returned value wraps a record call, when a record call comes earlier in a
// block that encloses the return, or when a deferred call records errors.
func checkReturns(pass *analysis.Pass, fn *ast.FuncDecl, spanPos token.Pos) {
	var records []record
	var returns []*ast.ReturnStmt
	deferred := false

	ast.PreorderStack(fn.Body, nil, func(n ast.Node, stack []ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return inDefer(stack)
		case *ast.ReturnStmt:
			if !inFuncLit(stack) {
				returns = append(returns, n)
			}
		case *ast.CallExpr:
			if !isRecordCall(pass, n) {
				return true
			}
			if inDefer(stack) {
				deferred = true
				return true
			}
			records = append(records, record{pos: n.Pos(), scope: innermostScope(stack)})
		}
		return true
	})
	if deferred {
		return
	}

	for _, ret := range returns {
		if ret.Pos() < spanPos || !returnsOriginError(pass, ret) || containsRecord(pass, ret) {
			continue
		}
		if coveredBy(records, ret) {
			continue
		}
		pass.Report(analysis.Diagnostic{
			Pos: ret.Pos(),
			Message: fmt.Sprintf("%s creates this error but does not record it; record it where it originates with "+
				"telemetry.RecordError (or RecordErrorAt<Level> to set the alert level)", fn.Name.Name),
		})
	}
}

// returnsOriginError reports whether ret returns an error created right here:
// errors.New, fmt.Errorf without %w, or errors.Flag on such an error. Errors
// passed up from a callee were recorded where they started, so recording
// them again would duplicate the exception event.
func returnsOriginError(pass *analysis.Pass, ret *ast.ReturnStmt) bool {
	for _, res := range ret.Results {
		if isOriginError(pass, res) {
			return true
		}
	}
	return false
}

func isOriginError(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg := moovutil.SelectorPackagePath(pass, sel)
	isErrors := pkg == "errors" || moovutil.IsErrorsPackage(pkg)
	switch {
	case sel.Sel.Name == "New" && isErrors:
		return true
	case sel.Sel.Name == "Errorf" && pkg == "fmt":
		if len(call.Args) == 0 {
			return false
		}
		tv := pass.TypesInfo.Types[call.Args[0]]
		return tv.Value != nil && tv.Value.Kind() == constant.String && !strings.Contains(constant.StringVal(tv.Value), "%w")
	case sel.Sel.Name == "Flag" && moovutil.IsErrorsPackage(pkg):
		return len(call.Args) > 0 && isOriginError(pass, call.Args[0])
	}
	return false
}

func coveredBy(records []record, ret *ast.ReturnStmt) bool {
	for _, r := range records {
		if r.pos < ret.Pos() && r.scope.Pos() <= ret.Pos() && ret.End() <= r.scope.End() {
			return true
		}
	}
	return false
}

func inDefer(stack []ast.Node) bool {
	for _, n := range stack {
		if _, ok := n.(*ast.DeferStmt); ok {
			return true
		}
	}
	return false
}

func inFuncLit(stack []ast.Node) bool {
	for _, n := range stack {
		if _, ok := n.(*ast.FuncLit); ok {
			return true
		}
	}
	return false
}

func innermostScope(stack []ast.Node) ast.Node {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].(type) {
		case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
			return stack[i]
		}
	}
	return stack[0]
}

func containsRecord(pass *analysis.Pass, ret *ast.ReturnStmt) bool {
	found := false
	ast.Inspect(ret, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && isRecordCall(pass, call) {
			found = true
		}
		return !found
	})
	return found
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

// spanStart returns the position of the first span created in fn, or
// token.NoPos when fn creates no span.
func spanStart(pass *analysis.Pass, fn *ast.FuncDecl) token.Pos {
	pos := token.NoPos
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || pos.IsValid() {
			return !pos.IsValid()
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		if (name == "StartSpan" || name == "StartLinkedRootSpan") &&
			moovutil.IsTelemetryPackage(moovutil.SelectorPackagePath(pass, sel)) {
			pos = call.Pos()
		}
		return true
	})
	return pos
}

func isRecordCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	name := sel.Sel.Name
	pkgPath := moovutil.SelectorPackagePath(pass, sel)
	if moovutil.IsTelemetryPackage(pkgPath) && strings.HasPrefix(name, "RecordError") {
		return true
	}
	return (name == "RecordError" || name == "SetStatus") &&
		(pkgPath == "go.opentelemetry.io/otel/trace" || strings.HasSuffix(pkgPath, "/otel/trace"))
}
