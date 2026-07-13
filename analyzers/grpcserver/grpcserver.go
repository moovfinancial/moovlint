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
			if !hasGRPCHandlerMethods(pass, typeDecl) {
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

func hasGRPCHandlerMethods(pass *analysis.Pass, typeSpec *ast.TypeSpec) bool {
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
		method := methodSet.At(i)
		fn, ok := method.Obj().(*types.Func)
		if !ok {
			continue
		}
		if isGRPCHandlerSignature(fn) {
			return true
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
