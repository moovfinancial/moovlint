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
	Doc:  "checks that exported methods on service/repo structs starting with a context parameter also start a span, excluding methods whose work is already auto-instrumented or that continue using an existing span",
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

			// A method whose only traced work is already auto-instrumented
			// (event production, database calls) must NOT open a span of its
			// own: that duplicates the library's span and fragments the trace.
			// See the engingeering-guide's telemetry doc, "Common Pitfalls".
			if wrapsOnlyAutoInstrumentedWork(pass, funcDecl) {
				return true
			}

			// The author already instrumented this method the way the guide
			// prescribes for a non-entry point: enrich the existing span
			// instead of opening a child span "just to annotate work".
			if annotatesAmbientSpan(pass, funcDecl) {
				return true
			}

			pass.Report(analysis.Diagnostic{
				Pos: funcDecl.Pos(),
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

// wrapsOnlyAutoInstrumentedWork reports whether every context-carrying call in
// the function body targets a library that already emits its own span. Such a
// method is a thin pass-through — the trace already covers its work, so
// requiring a manual span here would create the duplicate/unnecessary child
// span the telemetry guide warns against. Attributes worth recording belong on
// the surrounding span via telemetry.SetAttributes(ctx, ...).
//
// A body with no context-carrying calls at all is not exempt: it performs no
// work this analyzer can attribute to a library, so the entry-point
// expectation still applies.
func wrapsOnlyAutoInstrumentedWork(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	instrumented := 0
	untraced := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if untraced {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !passesContext(pass, call) {
			return true
		}
		switch pkgPath := calleePackagePath(pass, call); {
		case moovutil.IsAutoInstrumentedPackage(pkgPath):
			instrumented++
		case moovutil.IsTelemetryPackage(pkgPath), moovutil.IsMoovLogPackage(pkgPath), pkgPath == "context":
			// Annotation and context plumbing, not a unit of work.
		default:
			// Work this analyzer cannot show is already traced.
			untraced = true
		}
		return true
	})

	return instrumented > 0 && !untraced
}

// passesContext reports whether any argument of the call is a context.Context.
func passesContext(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if t := pass.TypesInfo.TypeOf(arg); t != nil && moovutil.IsContextType(t) {
			return true
		}
	}
	return false
}

// calleePackagePath resolves the import path of the package declaring the
// called function or method. Returns "" when it cannot be determined.
func calleePackagePath(pass *analysis.Pass, call *ast.CallExpr) string {
	var ident *ast.Ident
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.SelectorExpr:
		ident = fun.Sel
	case *ast.Ident:
		ident = fun
	default:
		return ""
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil || obj.Pkg() == nil {
		return ""
	}
	return obj.Pkg().Path()
}

// ambientSpanAnnotators are the calls that add to the span already in the
// context. Using one is a deliberate choice of the guide's recommended
// alternative to StartSpan at a non-entry point, so the method should not also
// be told to open a child span.
//
// RecordError is intentionally absent: it appears on nearly every error path,
// including in methods that genuinely are entry points, so treating it as an
// instrumentation signal would silence the rule almost everywhere.
var ambientSpanAnnotators = map[string]bool{
	"SetAttributes":   true,
	"AddEvent":        true,
	"SpanFromContext": true,
}

// annotatesAmbientSpan reports whether the function enriches the span already
// present in the context rather than starting one of its own.
func annotatesAmbientSpan(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok || !ambientSpanAnnotators[sel.Sel.Name] {
			return true
		}
		pkgPath := moovutil.SelectorPackagePath(pass, sel)
		if moovutil.IsTelemetryPackage(pkgPath) || pkgPath == "go.opentelemetry.io/otel/trace" {
			found = true
			return false
		}
		return true
	})
	return found
}
