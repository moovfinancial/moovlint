package blankdiscard

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "blankdiscard",
	Doc:  "detects blank identifier assignments (`_ = f()`) where the discarded value implements the error interface",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsMoovPackage(pass.Pkg.Path()) {
		return nil, nil
	}

	// Build error interface type for comparison
	errorInterface := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

	for _, file := range pass.Files {
		if moovutil.IsTestFile(pass.Fset.Position(file.Package).Filename) {
			continue
		}

		// Collect comments with their positions for checking adjacent comments
		comments := collectComments(file, pass.Fset)

		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			// Only check simple assignments (=), not short declarations (:=)
			if assign.Tok != token.ASSIGN {
				return true
			}

			// Only flag function/method calls, not variable assignments
			if len(assign.Rhs) != 1 {
				return true
			}
			if _, ok := assign.Rhs[0].(*ast.CallExpr); !ok {
				return true
			}

			// Check each assignment position for blank identifiers discarding errors
			for i, lhs := range assign.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != "_" {
					continue
				}

				// Get the corresponding RHS expression's type
				var rhsType types.Type
				if len(assign.Rhs) == 1 {
					// Tuple assignment: _, _ = f() or x, _ = f()
					rhsExpr := assign.Rhs[0]
					tv, ok := pass.TypesInfo.Types[rhsExpr]
					if !ok {
						continue
					}
					// Check if it's a tuple type
					if tuple, ok := tv.Type.(*types.Tuple); ok {
						if i < tuple.Len() {
							rhsType = tuple.At(i).Type()
						}
					} else if i == 0 {
						// Single value assignment: _ = f()
						rhsType = tv.Type
					}
				} else if i < len(assign.Rhs) {
					// Parallel assignment: _, _ = a, b
					tv, ok := pass.TypesInfo.Types[assign.Rhs[i]]
					if !ok {
						continue
					}
					rhsType = tv.Type
				}

				if rhsType == nil {
					continue
				}

				// Check if the type implements error interface
				if !types.Implements(rhsType, errorInterface) {
					continue
				}

				// Get the function/method name for the diagnostic
				funcName := getFuncName(assign.Rhs)

				// Check for Close() method with explanatory comment
				if funcName == "Close" && hasExplanatoryComment(assign, comments, pass.Fset) {
					continue
				}

				pass.Report(analysis.Diagnostic{
					Pos:     assign.Pos(),
					Message: fmt.Sprintf("discarding error from %s via blank assignment; handle, wrap-and-return, or add explanatory comment for Close() calls", funcName),
				})
			}
			return true
		})
	}
	return nil, nil
}

// commentInfo contains comment text and position
type commentInfo struct {
	line int
	text string
}

// collectComments extracts all comments from the file with their line numbers
func collectComments(file *ast.File, fset *token.FileSet) []commentInfo {
	var comments []commentInfo
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			pos := fset.Position(c.Pos())
			comments = append(comments, commentInfo{
				line: pos.Line,
				text: c.Text,
			})
		}
	}
	return comments
}

// hasExplanatoryComment checks if there's an explanatory comment on the same line or preceding line
// It excludes "// want" comments which are test annotations
func hasExplanatoryComment(assign *ast.AssignStmt, comments []commentInfo, fset *token.FileSet) bool {
	assignLine := fset.Position(assign.Pos()).Line
	for _, c := range comments {
		if c.line == assignLine || c.line == assignLine-1 {
			// Skip "// want" test annotations
			if strings.Contains(c.text, "// want") {
				continue
			}
			return true
		}
	}
	return false
}

// getFuncName extracts the function/method name from a single-element RHS.
// Callers must ensure rhs contains exactly one CallExpr.
func getFuncName(rhs []ast.Expr) string {
	call := rhs[0].(*ast.CallExpr)
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.IndexExpr:
		// Generic function: someFunc[T]()
		if ident, ok := fun.X.(*ast.Ident); ok {
			return ident.Name
		}
		if sel, ok := fun.X.(*ast.SelectorExpr); ok {
			return sel.Sel.Name
		}
	case *ast.IndexListExpr:
		// Generic function with multiple type params: someFunc[T, U]()
		if ident, ok := fun.X.(*ast.Ident); ok {
			return ident.Name
		}
		if sel, ok := fun.X.(*ast.SelectorExpr); ok {
			return sel.Sel.Name
		}
	}
	return "function"
}
