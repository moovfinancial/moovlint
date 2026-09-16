package contextcancel

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "contextcancel",
	Doc:  "checks that context.WithCancel/WithTimeout/WithDeadline results have a corresponding defer cancel()",
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

			cancelVars := findContextWithCalls(pass, fn)
			if len(cancelVars) == 0 {
				return true
			}

			deferredCancels := findDeferredCancels(fn)
			for _, cv := range cancelVars {
				if deferredCancels[cv.name] || managedElsewhere(fn, cv) {
					continue
				}
				pass.Report(analysis.Diagnostic{
					Pos:     fn.Pos(),
					Message: fmt.Sprintf("context cancel function '%s' is never deferred; add defer %s()", cv.name, cv.name),
				})
			}
			return true
		})
	}
	return nil, nil
}

func findContextWithCalls(pass *analysis.Pass, fn *ast.FuncDecl) []cancelVar {
	var cancelVars []cancelVar
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
			if name != "WithCancel" && name != "WithTimeout" && name != "WithDeadline" {
				continue
			}
			pkgPath := moovutil.SelectorPackagePath(pass, sel)
			if pkgPath != "context" {
				continue
			}
			if len(assign.Lhs) >= 2 {
				if id, ok := assign.Lhs[1].(*ast.Ident); ok && id.Name != "_" {
					cancelVars = append(cancelVars, cancelVar{name: id.Name, defPos: id.Pos()})
				}
			}
		}
		return true
	})
	return cancelVars
}

// cancelVar names a cancel function and the position where it is defined.
type cancelVar struct {
	name   string
	defPos token.Pos
}

// managedElsewhere reports whether the cancel function is referenced beyond
// its defining assignment, excluding a blank discard. A cancel value that is
// returned, stored, or captured by a shutdown closure is managed by its
// receiver, not leaked.
func managedElsewhere(fn *ast.FuncDecl, cv cancelVar) bool {
	blankDiscards := map[token.Pos]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			if i >= len(assign.Lhs) {
				break
			}
			if id, ok := rhs.(*ast.Ident); ok && id.Name == cv.name {
				if blank, ok := assign.Lhs[i].(*ast.Ident); ok && blank.Name == "_" {
					blankDiscards[id.Pos()] = true
				}
			}
		}
		return true
	})

	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		if id, ok := n.(*ast.Ident); ok && id.Name == cv.name && id.Pos() != cv.defPos && !blankDiscards[id.Pos()] {
			found = true
			return false
		}
		return true
	})
	return found
}

func findDeferredCancels(fn *ast.FuncDecl) map[string]bool {
	cancels := make(map[string]bool)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		call := deferStmt.Call
		if id, ok := call.Fun.(*ast.Ident); ok {
			cancels[id.Name] = true
		}
		return true
	})
	return cancels
}
