package grpcstatus

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "grpcstatus",
	Doc:  "checks that gRPC handler methods return errors through GrpcErrorStatus",
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
			if !isGRPCHandlerMethod(pass, fn) {
				return true
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				ret, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				if len(ret.Results) == 0 {
					return true
				}
				lastResult := ret.Results[len(ret.Results)-1]
				if isNilError(lastResult) {
					return true
				}
				if isGrpcErrorStatusCall(lastResult) {
					return true
				}
				pass.Report(analysis.Diagnostic{
					Pos:     ret.Pos(),
					Message: "gRPC handler error must be returned through GrpcErrorStatus(logger, err)",
				})
				return true
			})
			return true
		})
	}
	return nil, nil
}

func isGRPCHandlerMethod(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	recvObj := moovutil.ReceiverTypeName(pass, fn)
	if recvObj == nil {
		return false
	}
	named, ok := recvObj.Type().(*types.Named)
	if !ok {
		return false
	}
	return structEmbedsUnimplementedServer(named)
}

func structEmbedsUnimplementedServer(named *types.Named) bool {
	under, ok := named.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < under.NumFields(); i++ {
		field := under.Field(i)
		if !field.Embedded() {
			continue
		}
		name := field.Name()
		if strings.HasPrefix(name, "Unimplemented") && strings.HasSuffix(name, "Server") {
			return true
		}
	}
	return false
}

func isNilError(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

func isGrpcErrorStatusCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fun.Sel.Name == "GrpcErrorStatus"
	case *ast.Ident:
		return fun.Name == "GrpcErrorStatus"
	}
	return false
}
