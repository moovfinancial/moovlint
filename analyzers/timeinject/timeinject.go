package timeinject

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "timeinject",
	Doc:  "detects time.Now() calls in service methods that have a stime.TimeService field on their receiver",
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
	}
	return nil, nil
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
