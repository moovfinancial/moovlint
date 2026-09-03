package spanname

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var kebabRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var Analyzer = &analysis.Analyzer{
	Name: "spanname",
	Doc:  "checks span names passed to telemetry.StartSpan/StartLinkedRootSpan and span.SetName are lower-kebab-case",
	Run:  run,
}

func spanNameArgIndex(name string) (int, bool) {
	switch name {
	case "StartSpan", "StartLinkedRootSpan":
		return 1, true
	case "SetName":
		return 0, true
	}
	return 0, false
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
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			name := sel.Sel.Name
			if !moovutil.IsTelemetryPackage(moovutil.SelectorPackagePath(pass, sel)) {
				return true
			}
			argIdx, ok := spanNameArgIndex(name)
			if !ok {
				return true
			}
			if argIdx >= len(call.Args) {
				return true
			}
			lit, ok := call.Args[argIdx].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			if !kebabRe.MatchString(value) {
				pass.Report(analysis.Diagnostic{
					Pos:     call.Pos(),
					Message: fmt.Sprintf("span name %q must be lower-kebab-case (e.g. cron-handler)", value),
				})
			}
			return true
		})
	}
	return nil, nil
}
