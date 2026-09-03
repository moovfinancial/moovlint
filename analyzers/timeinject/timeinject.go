package timeinject

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "timeinject",
	Doc:  "detects time.Now() in service methods with a stime.TimeService field, and time.Now passed as a clock value instead of an injected clock",
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
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			recvObj := moovutil.ReceiverTypeName(pass, fn)
			if recvObj == nil {
				return true
			}
			if !hasTimeServiceField(pass, recvObj) {
				return true
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if sel.Sel.Name != "Now" {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || pkgIdent.Name != "time" {
					return true
				}
				pass.Report(analysis.Diagnostic{
					Pos:     call.Pos(),
					Message: "use injected stime.TimeService instead of time.Now() in service code with a time service field",
				})
				return true
			})
			return true
		})

		checkTimeNowAsValue(pass, file)
	}
	return nil, nil
}

// checkTimeNowAsValue flags the time.Now function passed around as a value
// (constructor args, field assignments, variables) instead of an injected clock.
func checkTimeNowAsValue(pass *analysis.Pass, file *ast.File) {
	called := make(map[token.Pos]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && isTimeNow(pass, sel) {
			called[sel.Pos()] = true
		}
		return true
	})

	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || !isTimeNow(pass, sel) {
			return true
		}
		if called[sel.Pos()] {
			return true
		}
		pass.Report(analysis.Diagnostic{
			Pos:     sel.Pos(),
			Message: "time.Now is passed as the wall clock; inject an stime.TimeService (or a stubbable func() time.Time parameter) instead",
		})
		return true
	})
}

func isTimeNow(pass *analysis.Pass, sel *ast.SelectorExpr) bool {
	if sel.Sel.Name != "Now" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "time" && moovutil.SelectorPackagePath(pass, sel) == "time"
}

func hasTimeServiceField(pass *analysis.Pass, typeName *types.TypeName) bool {
	named, ok := typeName.Type().(*types.Named)
	if !ok {
		return false
	}
	under, ok := named.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < under.NumFields(); i++ {
		field := under.Field(i)
		if isTimeServiceType(field.Type()) {
			return true
		}
	}
	return false
}

func isTimeServiceType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	pkgPath := obj.Pkg().Path()
	return pkgPath == "github.com/moov-io/base/stime" ||
		strings.HasSuffix(pkgPath, "/go-libs/stime") ||
		strings.HasSuffix(pkgPath, "/stime")
}
