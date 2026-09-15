package modelplacement

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/moovfinancial/moovlint/internal/placement"
	"golang.org/x/tools/go/analysis"
)

type Config struct {
	Enabled             bool   `json:"enabled"`
	ModelFiles          string `json:"model-files"`
	ImplementationFiles string `json:"implementation-files"`
	ModelTypes          string `json:"model-types"`
}

func New(cfg Config) *analysis.Analyzer {
	if cfg.ModelFiles == "" {
		cfg.ModelFiles = `^models?(_.*)?\.go$`
	}
	if cfg.ImplementationFiles == "" {
		cfg.ImplementationFiles = `^(service|repository|repo)(_.+)?\.go$`
	}
	if cfg.ModelTypes == "" {
		cfg.ModelTypes = `(Request|Response|Row|Record|Result|Query|Outcome|Page)$`
	}
	a := &analysis.Analyzer{
		Name: "modelplacement",
		Doc:  "flags exported data models in service and repository files (opt-in)",
	}
	a.Flags.BoolVar(&cfg.Enabled, "enabled", cfg.Enabled, "enable advisory model placement checks")
	a.Flags.StringVar(&cfg.ModelFiles, "model-files", cfg.ModelFiles, "regexp matching model file names")
	a.Flags.StringVar(&cfg.ImplementationFiles, "implementation-files", cfg.ImplementationFiles, "regexp matching implementation file names")
	a.Flags.StringVar(&cfg.ModelTypes, "model-types", cfg.ModelTypes, "regexp matching data model type names")
	a.Run = func(pass *analysis.Pass) (any, error) {
		if !cfg.Enabled {
			return nil, nil
		}
		models, err := regexp.Compile(cfg.ModelFiles)
		if err != nil {
			return nil, fmt.Errorf("model-files: %w", err)
		}
		implementations, err := regexp.Compile(cfg.ImplementationFiles)
		if err != nil {
			return nil, fmt.Errorf("implementation-files: %w", err)
		}
		types, err := regexp.Compile(cfg.ModelTypes)
		if err != nil {
			return nil, fmt.Errorf("model-types: %w", err)
		}
		for _, file := range pass.Files {
			name := filepath.Base(pass.Fset.Position(file.Pos()).Filename)
			if strings.HasSuffix(name, "_test.go") || ast.IsGenerated(file) || models.MatchString(name) || !implementations.MatchString(name) {
				continue
			}
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Assign.IsValid() || !types.MatchString(ts.Name.Name) {
						continue
					}
					if placement.Model(pass.TypesInfo.TypeOf(ts.Name)) != nil {
						pass.Reportf(ts.Pos(), "data model %s is declared in %s; move it to a model file matching %s", ts.Name.Name, name, cfg.ModelFiles)
					}
				}
			}
		}
		return nil, nil
	}
	return a
}
