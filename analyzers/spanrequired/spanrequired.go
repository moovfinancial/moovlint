package spanrequired

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "spanrequired",
	Doc:  "checks that exported methods on service/repo structs starting with a context parameter also start a span, excluding thin wrappers around a single delegate, methods whose work (followed through same-package helpers) is already auto-instrumented, and methods that continue using an existing span",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	decls := funcDecls(pass)
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

			work := summarizeWork(pass, decls, funcDecl, map[*ast.FuncDecl]bool{})

			// A method whose only traced work is already auto-instrumented
			// (event production, database calls) must NOT open a span of its
			// own: that duplicates the library's span and fragments the trace.
			// See the engingeering-guide's telemetry doc, "Common Pitfalls".
			if work.wrapsOnlyAutoInstrumentedWork() {
				return true
			}

			// A method that only hands off to one delegate has no work of its
			// own worth a span; the guide calls that an unnecessary child span.
			if work.isThinWrapper() {
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

// workSummary counts the context-carrying calls a function makes, after
// following calls into helpers declared in the same package.
type workSummary struct {
	// instrumented counts calls into libraries that emit their own span, and
	// calls to same-package functions that start their own span.
	instrumented int
	// untraced counts calls whose work this analyzer cannot show is traced.
	untraced int
}

// summarizeWork walks fn's body and classifies every context-carrying call.
//
// Calls to functions declared in the analyzed package are followed, as are
// interface method calls with implementations in the package (a service
// calling its Repository interface reaches the SQL implementation): a helper
// that only reaches auto-instrumented libraries contributes that, so ex.
// templater's queryScan* -> db.QueryContext pattern counts as database
// work rather than as an opaque call. A helper whose context calls are only
// annotation or logging contributes nothing.
//
// Calls inside a loop count twice, since they run once per iteration and so
// can never be a single delegated call.
func summarizeWork(pass *analysis.Pass, decls map[*types.Func]*ast.FuncDecl, fn *ast.FuncDecl, visiting map[*ast.FuncDecl]bool) workSummary {
	var sum workSummary
	if fn.Body == nil || visiting[fn] {
		return sum
	}
	visiting[fn] = true
	defer delete(visiting, fn)

	var walk func(n ast.Node, weight int)
	walk = func(root ast.Node, weight int) {
		ast.Inspect(root, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ForStmt:
				if n != root {
					walk(n, 2*weight)
					return false
				}
			case *ast.RangeStmt:
				if n != root {
					walk(n, 2*weight)
					return false
				}
			case *ast.CallExpr:
				if !passesContext(pass, n) {
					return true
				}
				switch pkgPath := calleePackagePath(pass, n); {
				case moovutil.IsAutoInstrumentedPackage(pkgPath):
					sum.instrumented += weight
					return true
				case moovutil.IsTelemetryPackage(pkgPath), moovutil.IsMoovLogPackage(pkgPath), pkgPath == "context":
					// Annotation and context plumbing, not a unit of work.
					return true
				}
				impls := implementations(decls, calleeFunc(pass, n))
				if len(impls) == 0 {
					sum.untraced += weight
					return true
				}
				// An interface call may reach any implementation, so it
				// counts as the most expensive of them.
				var worst workSummary
				for _, impl := range impls {
					sub := workSummary{instrumented: 1}
					if !hasTelemetrySpanCall(pass, impl) {
						sub = summarizeWork(pass, decls, impl, visiting)
					}
					worst.instrumented = max(worst.instrumented, sub.instrumented)
					worst.untraced = max(worst.untraced, sub.untraced)
				}
				sum.instrumented += worst.instrumented * weight
				sum.untraced += worst.untraced * weight
			}
			return true
		})
	}
	walk(fn.Body, 1)
	return sum
}

// wrapsOnlyAutoInstrumentedWork reports whether every context-carrying call
// the method makes, directly or through same-package helpers, targets a
// library that already emits its own span. Such a method is a pass-through:
// the trace already covers its work, so requiring a manual span here would
// create the duplicate/unnecessary child span the telemetry guide warns
// against. Attributes worth recording belong on the surrounding span via
// telemetry.SetAttributes(ctx, ...).
//
// A body with no context-carrying calls at all is not exempt: it performs no
// work this analyzer can attribute to a library, so the entry-point
// expectation still applies.
func (w workSummary) wrapsOnlyAutoInstrumentedWork() bool {
	return w.instrumented > 0 && w.untraced == 0
}

// isThinWrapper reports whether the method delegates to exactly one unit of
// work it cannot see into (an interface, another package, a client) and does
// nothing else with the context. The span, if one is warranted, belongs to
// the delegate; a second span here would only nest an identical one around it.
func (w workSummary) isThinWrapper() bool {
	return w.instrumented == 0 && w.untraced == 1
}

// implementations returns the same-package declarations a call to fn can
// reach: fn itself, or for an interface method, every method of that name on
// a package type implementing the interface. Returns nil when none are
// declared in the package.
func implementations(decls map[*types.Func]*ast.FuncDecl, fn *types.Func) []*ast.FuncDecl {
	if fn == nil {
		return nil
	}
	if decl, ok := decls[fn]; ok {
		return []*ast.FuncDecl{decl}
	}
	recv := fn.Signature().Recv()
	if recv == nil {
		return nil
	}
	iface, ok := recv.Type().Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	var impls []*ast.FuncDecl
	for obj, decl := range decls {
		objRecv := obj.Signature().Recv()
		if objRecv == nil || obj.Name() != fn.Name() {
			continue
		}
		if types.Implements(objRecv.Type(), iface) || types.Implements(types.NewPointer(objRecv.Type()), iface) {
			impls = append(impls, decl)
		}
	}
	return impls
}

// funcDecls indexes the package's non-test function declarations by their
// object, so calls to same-package helpers and interface implementations can
// be followed. Test files are left out so a mock in the test variant of the
// package does not change the result.
func funcDecls(pass *analysis.Pass) map[*types.Func]*ast.FuncDecl {
	decls := make(map[*types.Func]*ast.FuncDecl)
	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if obj, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func); ok {
				decls[obj] = fn
			}
		}
	}
	return decls
}

// calleeFunc resolves the statically called function or method, or nil for
// calls through function values.
func calleeFunc(pass *analysis.Pass, call *ast.CallExpr) *types.Func {
	var ident *ast.Ident
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.SelectorExpr:
		ident = fun.Sel
	case *ast.Ident:
		ident = fun
	case *ast.IndexExpr:
		ident = genericIdent(fun.X)
	case *ast.IndexListExpr:
		ident = genericIdent(fun.X)
	}
	if ident == nil {
		return nil
	}
	fn, ok := pass.TypesInfo.Uses[ident].(*types.Func)
	if !ok {
		return nil
	}
	return fn.Origin()
}

func genericIdent(x ast.Expr) *ast.Ident {
	switch x := x.(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		return x.Sel
	}
	return nil
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
