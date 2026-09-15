package moneyfloat

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "moneyfloat",
	Doc:  "detects float32/float64 used for monetary values (fields and params named amount, balance, fee, or total)",
	Run:  run,
}

var moneySuffixes = []string{"amount", "balance", "fee", "fees", "total"}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {
			field, ok := n.(*ast.Field)
			if !ok {
				return true
			}
			t := pass.TypesInfo.TypeOf(field.Type)
			if t == nil || !isFloatType(t) {
				return true
			}
			for _, name := range field.Names {
				if !isMoneyName(name.Name) {
					continue
				}
				pass.Report(analysis.Diagnostic{
					Pos:     name.Pos(),
					Message: fmt.Sprintf("%s uses a float for a monetary value; use an integer minor-unit type or a decimal type", name.Name),
				})
			}
			return true
		})
	}
	return nil, nil
}

func isMoneyName(name string) bool {
	n := strings.ToLower(name)
	for _, suffix := range moneySuffixes {
		if strings.HasSuffix(n, suffix) {
			return true
		}
	}
	return false
}

func isFloatType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return basic.Info()&types.IsFloat != 0
}
