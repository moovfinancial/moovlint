package fixtureplacement

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"github.com/moovfinancial/moovlint/internal/placement"
	"golang.org/x/tools/go/analysis"
)

type Config struct {
	Enabled         bool   `json:"enabled"`
	FixturePackages string `json:"fixture-packages"`
}

func New(cfg Config) *analysis.Analyzer {
	if cfg.FixturePackages == "" {
		cfg.FixturePackages = `(^|/)(fixtures|testfixtures|testutil)(/|$)`
	}
	a := &analysis.Analyzer{
		Name: "fixtureplacement",
		Doc:  "flags test model builders outside configured fixture packages (opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable advisory fixture placement checks")
	a.Flags.StringVar(&cfg.FixturePackages, "fixture-packages", cfg.FixturePackages, "regexp matching fixture import paths")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		paths, err := regexp.Compile(cfg.FixturePackages)
		if err != nil {
			return nil, fmt.Errorf("fixture-packages: %w", err)
		}
		if paths.MatchString(strings.TrimSuffix(pass.Pkg.Path(), "_test")) {
			return nil, nil
		}
		for _, file := range pass.Files {
			if !strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") || ast.IsGenerated(file) {
				continue
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || fn.Recv != nil || fn.Type.Results == nil {
					continue
				}
				sig := pass.TypesInfo.ObjectOf(fn.Name).Type().(*types.Signature)
				results := sig.Results()
				if results.Len() == 0 || results.Len() > 2 || (results.Len() == 2 && !types.Identical(results.At(1).Type(), types.Universe.Lookup("error").Type())) {
					continue
				}
				model := placement.Model(results.At(0).Type())
				if model == nil || !inModule(pass, model.Obj().Pkg()) || !buildsModel(pass, fn.Body, model) {
					continue
				}
				pass.Reportf(fn.Pos(), "test helper %s builds %s; move the model builder to a fixture package matching %s", fn.Name.Name, model.Obj().Name(), cfg.FixturePackages)
			}
		}
		return nil, nil
	}
	return a
}

func inModule(pass *analysis.Pass, pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	if pkg.Path() == strings.TrimSuffix(pass.Pkg.Path(), "_test") || pkg == pass.Pkg {
		return true
	}
	return pass.Module != nil && (pkg.Path() == pass.Module.Path || strings.HasPrefix(pkg.Path(), pass.Module.Path+"/"))
}

func buildsModel(pass *analysis.Pass, body *ast.BlockStmt, model *types.Named) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		switch n := n.(type) {
		case *ast.CompositeLit:
			if len(n.Elts) > 0 && types.Identical(pass.TypesInfo.TypeOf(n), model) {
				found = true
			}
		case *ast.CallExpr:
			if returned := placement.Model(pass.TypesInfo.TypeOf(n)); returned != nil && types.Identical(returned, model) {
				found = true
			}
		}
		return !found
	})
	return found
}
