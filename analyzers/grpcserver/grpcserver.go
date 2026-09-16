package grpcserver

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "grpcserver",
	Doc:  "checks that gRPC controller structs embed their generated Unimplemented*Server type",
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
			typeDecl, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			structType, ok := typeDecl.Type.(*ast.StructType)
			if !ok {
				return true
			}
			if !implementsServerInterface(pass, typeDecl) {
				return true
			}
			if embedsUnimplementedServer(structType) {
				return true
			}
			pass.Report(analysis.Diagnostic{
				Pos:     typeDecl.Pos(),
				Message: fmt.Sprintf("gRPC controller %s must embed its Unimplemented*Server type for forward compatibility", typeDecl.Name.Name),
			})
			return true
		})
	}
	return nil, nil
}

// implementsServerInterface reports whether the struct implements, in whole or
// in part, a generated service interface from another package. A real gRPC
// controller has at least one method whose name matches a method of a
// cross-package interface named *Server, and whose request type comes from
// that same package. The plain handler-shape check (context.Context, T) (R,
// error) matches ordinary service and repository methods, so it cannot carry
// the rule alone.
func implementsServerInterface(pass *analysis.Pass, typeSpec *ast.TypeSpec) bool {
	obj := pass.TypesInfo.Defs[typeSpec.Name]
	if obj == nil {
		return false
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return false
	}
	methodSet := types.NewMethodSet(types.NewPointer(named))
	for i := 0; i < methodSet.Len(); i++ {
		fn, ok := methodSet.At(i).Obj().(*types.Func)
		if !ok || !isGRPCHandlerSignature(fn) {
			continue
		}
		if implementsServerMethod(pass, fn) {
			return true
		}
	}
	return false
}

// implementsServerMethod reports whether fn matches a method of a *Server
// interface from the package that defines fn's request type.
func implementsServerMethod(pass *analysis.Pass, fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 2 {
		return false
	}

	request := sig.Params().At(1).Type()
	if ptr, ok := request.(*types.Pointer); ok {
		request = ptr.Elem()
	}
	requestNamed, ok := request.(*types.Named)
	if !ok {
		return false
	}
	pkg := requestNamed.Obj().Pkg()
	if pkg == nil || pkg == pass.Pkg {
		return false
	}

	for _, name := range pkg.Scope().Names() {
		typeName, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok || !strings.HasSuffix(name, "Server") {
			continue
		}
		iface, ok := typeName.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		for i := 0; i < iface.NumMethods(); i++ {
			if iface.Method(i).Name() == fn.Name() {
				return true
			}
		}
	}
	return false
}

func isGRPCHandlerSignature(method *types.Func) bool {
	sig, ok := method.Type().(*types.Signature)
	if !ok {
		return false
	}
	if sig.Params().Len() != 2 {
		return false
	}
	firstParam := sig.Params().At(0)
	if firstParam.Type().String() != "context.Context" {
		return false
	}
	if sig.Results().Len() != 2 {
		return false
	}
	lastResult := sig.Results().At(1)
	return lastResult.Type().String() == "error"
}

func embedsUnimplementedServer(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if strings.HasPrefix(name.Name, "Unimplemented") && strings.HasSuffix(name.Name, "Server") {
				return true
			}
		}
		if field.Names == nil {
			ident, ok := field.Type.(*ast.Ident)
			if ok && strings.HasPrefix(ident.Name, "Unimplemented") && strings.HasSuffix(ident.Name, "Server") {
				return true
			}
			sel, ok := field.Type.(*ast.SelectorExpr)
			if ok && sel.Sel != nil && strings.HasPrefix(sel.Sel.Name, "Unimplemented") && strings.HasSuffix(sel.Sel.Name, "Server") {
				return true
			}
		}
	}
	return false
}
