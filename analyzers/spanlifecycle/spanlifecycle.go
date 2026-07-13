package spanlifecycle

import (
	"fmt"
	"go/ast"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "spanlifecycle",
	Doc:  "checks that spans created with telemetry.StartSpan or StartLinkedRootSpan are ended with defer span.End()",
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
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}

			spanVars := findSpanCreations(pass, fn)
			if len(spanVars) == 0 {
				return true
			}

			endedVars := findDeferredEnds(fn)
			for _, sv := range spanVars {
				if !endedVars[sv] {
					pass.Report(analysis.Diagnostic{
						Pos:     fn.Pos(),
						Message: fmt.Sprintf("span assigned to '%s' is never ended; add defer %s.End()", sv, sv),
					})
				}
			}
			return true
		})
	}
	return nil, nil
}

func findSpanCreations(pass *analysis.Pass, fn *ast.FuncDecl) []string {
	var spanVars []string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, rhs := range assign.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok {
				continue
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			name := sel.Sel.Name
			if name != "StartSpan" && name != "StartLinkedRootSpan" {
				continue
			}
			if !moovutil.IsTelemetryPackage(moovutil.SelectorPackagePath(pass, sel)) {
				continue
			}
			if name == "StartSpan" && len(assign.Rhs) == 1 && len(assign.Lhs) >= 2 {
				if id, ok := assign.Lhs[1].(*ast.Ident); ok && id.Name != "_" {
					spanVars = append(spanVars, id.Name)
				}
			} else if name == "StartLinkedRootSpan" && len(assign.Lhs) == 1 {
				if id, ok := assign.Lhs[0].(*ast.Ident); ok && id.Name != "_" {
					spanVars = append(spanVars, id.Name)
				}
			}
		}
		return true
	})
	return spanVars
}

func findDeferredEnds(fn *ast.FuncDecl) map[string]bool {
	ended := make(map[string]bool)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		call := deferStmt.Call
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "End" {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			ended[id.Name] = true
		}
		return true
	})
	return ended
}
