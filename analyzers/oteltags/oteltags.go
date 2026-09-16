package oteltags

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "oteltags",
	Doc:  "checks that otel struct tags use lower snake case and do not include omitempty",
	Run:  run,
}

var otelTagRe = regexp.MustCompile(`otel:"([^"]*)"`)

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsMoovPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			structType, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range structType.Fields.List {
				if field.Tag == nil {
					continue
				}
				tag := field.Tag.Value
				matches := otelTagRe.FindStringSubmatch(tag)
				if len(matches) < 2 {
					continue
				}
				otelValue := matches[1]

				// "-" is the skip marker; the telemetry library ignores the
				// field the same as an untagged one.
				if otelValue == "-" {
					continue
				}

				if strings.Contains(otelValue, "omitempty") {
					pass.Report(analysis.Diagnostic{
						Pos:     field.Pos(),
						Message: "otel tag must not include omitempty; span attributes should always be recorded",
					})
				}

				if !isLowerSnakeCase(otelValue) {
					pass.Report(analysis.Diagnostic{
						Pos:     field.Pos(),
						Message: fmt.Sprintf("otel tag %q must use lower snake case", otelValue),
					})
				}

				if hasBadType(pass, field) {
					pass.Report(analysis.Diagnostic{
						Pos:     field.Pos(),
						Message: "otel tag on field with map, slice-of-struct, or deeply nested type; use scalar attributes or bounded StringSlice instead",
					})
				}
			}

			return true
		})
	}
	return nil, nil
}

func isLowerSnakeCase(s string) bool {
	if s == "" {
		return true
	}
	parts := strings.Split(s, ",")
	name := parts[0]
	if name == "" {
		return true
	}
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			return false
		}
		if r == '-' {
			return false
		}
	}
	return true
}

func hasBadType(pass *analysis.Pass, field *ast.Field) bool {
	if len(field.Names) == 0 {
		return false
	}
	t := pass.TypesInfo.TypeOf(field.Type)
	if t == nil {
		return false
	}
	if serializesAsScalarAttribute(t) {
		return false
	}
	return isBadAttrType(t, 0)
}

// scalarAttributeTypes are the interfaces the telemetry library satisfies with
// one scalar attribute instead of recursing into the value.
var scalarAttributeTypes = func() []*types.Interface {
	method := func(name string, result types.Type) *types.Func {
		return types.NewFunc(token.NoPos, nil, name,
			types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", result)), false))
	}
	ifaces := make([]*types.Interface, 0, 4)
	for _, m := range []*types.Func{
		method("AttributeString", types.Typ[types.String]),
		method("Value", types.Typ[types.String]),
		method("Value", types.Typ[types.Int]),
		method("Value", types.Typ[types.Int64]),
	} {
		ifaces = append(ifaces, types.NewInterfaceType([]*types.Func{m}, nil).Complete())
	}
	return ifaces
}()

func serializesAsScalarAttribute(t types.Type) bool {
	if isTimeType(t) {
		return true
	}
	for _, iface := range scalarAttributeTypes {
		if types.Implements(t, iface) {
			return true
		}
	}
	return false
}

func isTimeType(t types.Type) bool {
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
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == "time" && obj.Name() == "Time"
}

func isBadAttrType(t types.Type, depth int) bool {
	if depth > 2 {
		return true
	}
	switch v := t.(type) {
	case *types.Named:
		under := v.Underlying()
		if _, ok := under.(*types.Struct); ok {
			return true
		}
		return isBadAttrType(under, depth)
	case *types.Map:
		return true
	case *types.Slice:
		return isBadAttrType(v.Elem(), depth+1)
	case *types.Array:
		return isBadAttrType(v.Elem(), depth+1)
	case *types.Pointer:
		under := v.Elem()
		if named, ok := under.(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok {
				return true
			}
		}
		return isBadAttrType(under, depth)
	}
	return false
}
