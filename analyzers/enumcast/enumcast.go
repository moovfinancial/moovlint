package enumcast

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "enumcast",
	Doc:  "detects unchecked conversions of raw strings to enum-like named string types outside validation and mapper functions",
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
			if isValidationOrMapper(fn.Name.Name) {
				continue
			}
			if funcHasSwitch(fn) {
				continue
			}
			checkFuncBody(pass, fn)
		}
	}
	return nil, nil
}

func checkFuncBody(pass *analysis.Pass, fn *ast.FuncDecl) {
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if len(call.Args) != 1 {
			return true
		}
		tv, ok := pass.TypesInfo.Types[call.Fun]
		if !ok || !tv.IsType() {
			return true
		}
		target, ok := tv.Type.(*types.Named)
		if !ok || !isEnumStringType(target) {
			return true
		}
		if funcHasMapperMap(pass, fn, target) {
			return true
		}
		if isKnownConstantValue(pass, call.Args[0], target) {
			return true
		}
		pass.Report(analysis.Diagnostic{
			Pos: call.Pos(),
			Message: fmt.Sprintf("unchecked conversion to enum type %s; validate the value against the known constants (switch, map lookup, or a validation function)",
				target.Obj().Name()),
		})
		return true
	})
}

func isEnumStringType(target *types.Named) bool {
	basic, ok := target.Underlying().(*types.Basic)
	if !ok || basic.Info()&types.IsString == 0 {
		return false
	}
	pkg := target.Obj().Pkg()
	if pkg == nil {
		return false
	}
	consts := 0
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		if c, ok := obj.(*types.Const); ok && types.Identical(c.Type(), target) {
			consts++
		}
	}
	return consts >= 2
}

func isKnownConstantValue(pass *analysis.Pass, arg ast.Expr, target *types.Named) bool {
	tv, ok := pass.TypesInfo.Types[arg]
	if !ok || tv.Value == nil {
		return false
	}
	pkg := target.Obj().Pkg()
	if pkg == nil {
		return false
	}
	value := tv.Value.ExactString()
	for _, name := range pkg.Scope().Names() {
		if c, ok := pkg.Scope().Lookup(name).(*types.Const); ok {
			if types.Identical(c.Type(), target) && c.Val().ExactString() == value {
				return true
			}
		}
	}
	return false
}

func isValidationOrMapper(name string) bool {
	n := strings.ToLower(name)
	for _, part := range []string{"valid", "map", "parse", "convert"} {
		if strings.Contains(n, part) {
			return true
		}
	}
	return false
}

func funcHasSwitch(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.SwitchStmt); ok {
			found = true
			return false
		}
		if _, ok := n.(*ast.TypeSwitchStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func funcHasMapperMap(pass *analysis.Pass, fn *ast.FuncDecl, target *types.Named) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		t := pass.TypesInfo.TypeOf(lit)
		if t == nil {
			return true
		}
		if mt, ok := t.Underlying().(*types.Map); ok && types.Identical(mt.Elem(), target) {
			found = true
			return false
		}
		return true
	})
	return found
}
