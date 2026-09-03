package uuidgen

import (
	"go/ast"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var bannedUUIDFuncs = map[string]bool{
	"New":               true,
	"NewString":         true,
	"NewRandom":         true,
	"NewRandomFromTime": true,
	"NewV6":             true,
	"NewV7":             true,
}

var Analyzer = &analysis.Analyzer{
	Name: "uuidgen",
	Doc:  "detects uuid.New* used for ID generation in mid-based services; use mid.NewRandomID so entity IDs carry their type",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}
	if !usesMid(pass) && !strings.HasPrefix(pass.Pkg.Path(), "github.com/moovfinancial/") {
		return nil, nil
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !bannedUUIDFuncs[sel.Sel.Name] {
				return true
			}
			if moovutil.SelectorPackagePath(pass, sel) != "github.com/google/uuid" {
				return true
			}
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				Message: "uuid." + sel.Sel.Name + " generates untyped IDs; use mid.NewRandomID[T] so entity IDs carry their type",
			})
			return true
		})
	}
	return nil, nil
}

func usesMid(pass *analysis.Pass) bool {
	for _, imp := range pass.Pkg.Imports() {
		if moovutil.IsMidPackage(imp.Path()) {
			return true
		}
	}
	return false
}
