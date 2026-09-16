package mapderef

import (
	"go/ast"
	"go/types"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

type Config struct {
	Enabled bool `json:"enabled"`
}

func New(cfg Config) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "mapderef",
		Doc:  "detects m[k].Field dereferences on maps of pointers or interfaces without a comma-ok check; a missing key yields nil (advisory, opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable advisory map dereference checks")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		return run(pass)
	}
	return a
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.SelectorExpr:
				if idx, ok := node.X.(*ast.IndexExpr); ok && isNilableElem(pass, idx) {
					report(pass, idx)
				}
			case *ast.StarExpr:
				if idx, ok := node.X.(*ast.IndexExpr); ok && isNilableElem(pass, idx) {
					report(pass, idx)
				}
			}
			return true
		})
	}
	return nil, nil
}

func isNilableElem(pass *analysis.Pass, idx *ast.IndexExpr) bool {
	t := pass.TypesInfo.TypeOf(idx)
	if t == nil {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return true
	}
	return false
}

func report(pass *analysis.Pass, idx *ast.IndexExpr) {
	pass.Report(analysis.Diagnostic{
		Pos:     idx.Pos(),
		Message: "map lookup is dereferenced without a comma-ok check; a missing key yields a nil value, use v, ok := m[k]",
	})
}
