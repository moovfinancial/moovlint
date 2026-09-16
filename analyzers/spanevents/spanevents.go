package spanevents

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "spanevents",
	Doc:  "detects logger.Info().Log() and logger.Warn().Log() calls in service/repo code and suggests telemetry.AddEvent or telemetry.RecordError",
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
			if !ok || fn.Body == nil {
				continue
			}
			walk(pass, fn.Body, hasContextParam(pass, fn.Type.Params))
		}
	}
	return nil, nil
}

// walk reports logger.Info/Warn().Log calls, tracking whether a context is in
// scope. Without one the telemetry suggestion cannot be applied, so boot and
// shutdown helpers without a context parameter are left alone.
func walk(pass *analysis.Pass, body ast.Node, hasCtx bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		if lit, ok := n.(*ast.FuncLit); ok && lit.Body != nil {
			walk(pass, lit.Body, hasCtx || hasContextParam(pass, lit.Type.Params))
			return false
		}
		if !hasCtx {
			return true
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Log" && sel.Sel.Name != "Logf") {
			return true
		}

		infoCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		infoSel, ok := infoCall.Fun.(*ast.SelectorExpr)
		if !ok || (infoSel.Sel.Name != "Info" && infoSel.Sel.Name != "Warn") {
			return true
		}

		rootExpr := infoSel.X
		tv, ok := pass.TypesInfo.Types[rootExpr]
		if !ok {
			return true
		}

		if !isMoovLogger(tv.Type) {
			return true
		}

		level := infoSel.Sel.Name

		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			Message: fmt.Sprintf("use telemetry.AddEvent or telemetry.RecordError instead of logger.%s().Log", level),
		})
		return true
	})
}

// hasContextParam reports whether the parameter list declares a context.Context.
func hasContextParam(pass *analysis.Pass, params *ast.FieldList) bool {
	if params == nil {
		return false
	}
	for _, field := range params.List {
		t := pass.TypesInfo.TypeOf(field.Type)
		if t == nil {
			continue
		}
		if named, ok := t.(*types.Named); ok {
			if obj := named.Obj(); obj != nil && obj.Pkg() != nil &&
				obj.Pkg().Path() == "context" && obj.Name() == "Context" {
				return true
			}
		}
	}
	return false
}

func isMoovLogger(t types.Type) bool {
	if t == nil {
		return false
	}
	for {
		ptr, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	if named.Obj().Pkg() == nil {
		return false
	}
	pkg := named.Obj().Pkg().Path()
	return moovutil.IsMoovLogPackage(pkg) ||
		strings.HasSuffix(pkg, "moov-io/base/log") ||
		strings.HasSuffix(pkg, "moovfinancial/go-libs/observability/log")
}
