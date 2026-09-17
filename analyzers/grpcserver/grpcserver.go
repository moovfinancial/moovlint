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
			expectedEmbed := unimplementedServerEmbed(pass, typeDecl)
			if expectedEmbed == nil {
				return true
			}
			if embedsUnimplementedServer(pass, structType, expectedEmbed) {
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

// unimplementedServerEmbed returns the generated type a controller must embed.
// A matching method name alone is insufficient: storage code can accept a
// protobuf request without implementing the generated server method.
func unimplementedServerEmbed(pass *analysis.Pass, typeSpec *ast.TypeSpec) *types.Named {
	obj := pass.TypesInfo.Defs[typeSpec.Name]
	if obj == nil {
		return nil
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}
	methodSet := types.NewMethodSet(types.NewPointer(named))
	for i := 0; i < methodSet.Len(); i++ {
		fn, ok := methodSet.At(i).Obj().(*types.Func)
		if !ok {
			continue
		}
		if embed := unimplementedServerMethodEmbed(pass, fn); embed != nil {
			return embed
		}
	}
	return nil
}

// unimplementedServerMethodEmbed returns the generated Unimplemented*Server
// type when fn exactly matches a method on its generated *Server interface.
func unimplementedServerMethodEmbed(pass *analysis.Pass, fn *types.Func) *types.Named {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 2 {
		return nil
	}

	request := sig.Params().At(1).Type()
	if ptr, ok := request.(*types.Pointer); ok {
		request = ptr.Elem()
	}
	requestNamed, ok := request.(*types.Named)
	if !ok {
		return nil
	}
	pkg := requestNamed.Obj().Pkg()
	if pkg == nil || pkg == pass.Pkg {
		return nil
	}

	for _, name := range pkg.Scope().Names() {
		typeName, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok || !strings.HasSuffix(name, "Server") || strings.HasPrefix(name, "Unimplemented") {
			continue
		}
		iface, ok := typeName.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		for i := 0; i < iface.NumMethods(); i++ {
			method := iface.Method(i)
			if method.Name() != fn.Name() || !sameSignature(sig, method.Type().(*types.Signature)) {
				continue
			}
			embedName := "Unimplemented" + name
			embed, ok := pkg.Scope().Lookup(embedName).(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := embed.Type().(*types.Named)
			if ok {
				return named
			}
		}
	}
	return nil
}

func sameSignature(left, right *types.Signature) bool {
	if left.Variadic() != right.Variadic() || left.Params().Len() != right.Params().Len() || left.Results().Len() != right.Results().Len() {
		return false
	}
	for i := 0; i < left.Params().Len(); i++ {
		if !types.Identical(left.Params().At(i).Type(), right.Params().At(i).Type()) {
			return false
		}
	}
	for i := 0; i < left.Results().Len(); i++ {
		if !types.Identical(left.Results().At(i).Type(), right.Results().At(i).Type()) {
			return false
		}
	}
	return true
}

func embedsUnimplementedServer(pass *analysis.Pass, structType *ast.StructType, expected *types.Named) bool {
	for _, field := range structType.Fields.List {
		if len(field.Names) != 0 {
			continue
		}
		fieldType := pass.TypesInfo.TypeOf(field.Type)
		if ptr, ok := fieldType.(*types.Pointer); ok {
			fieldType = ptr.Elem()
		}
		if types.Identical(fieldType, expected) {
			return true
		}
	}
	return false
}
