package nolintguard

import (
	"go/ast"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "nolintguard",
	Doc:  "checks that //nolint directives target a specific linter and include an explanation",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsMoovPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, cg := range file.Comments {
			for _, c := range cg.List {
				checkNolintDirective(pass, c)
			}
		}
	}
	return nil, nil
}

func checkNolintDirective(pass *analysis.Pass, c *ast.Comment) {
	text := c.Text
	if !strings.HasPrefix(text, "//nolint") {
		return
	}

	rest := strings.TrimPrefix(text, "//nolint")
	if rest == "" || !strings.HasPrefix(rest, ":") {
		pass.Report(analysis.Diagnostic{
			Pos:     c.Pos(),
			Message: "bare //nolint without a linter name is not allowed; specify the linter (e.g. //nolint:errcheck)",
		})
		return
	}

	after := strings.TrimPrefix(text, "//nolint:")
	parts := strings.SplitN(after, "//", 2)
	linterList := strings.TrimSpace(parts[0])

	for _, l := range strings.Split(linterList, ",") {
		if strings.TrimSpace(l) == "all" {
			pass.Report(analysis.Diagnostic{
				Pos:     c.Pos(),
				Message: "//nolint:all is not allowed; target the specific linter",
			})
			return
		}
	}

	explanation := ""
	if len(parts) > 1 {
		explanation = strings.TrimSpace(parts[1])
	}
	if explanation == "" || strings.HasPrefix(explanation, "want ") || strings.HasPrefix(explanation, "want\"") {
		pass.Report(analysis.Diagnostic{
			Pos:     c.Pos(),
			Message: "//nolint directive requires an explanation comment on the same line",
		})
	}
}
