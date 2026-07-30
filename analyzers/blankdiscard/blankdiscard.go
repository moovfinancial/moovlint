package blankdiscard

import (
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
			if len(assign.Rhs) != 1 && len(assign.Rhs) != len(assign.Lhs) {
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
				var rhsCall *ast.CallExpr
				if len(assign.Rhs) == 1 {
					// Tuple assignment: _, _ = f() or x, _ = f()
					rhsExpr := assign.Rhs[0]
					call, ok := rhsExpr.(*ast.CallExpr)
					if !ok {
						continue
					}
					rhsCall = call
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
				} else if len(assign.Rhs) == len(assign.Lhs) {
					// Parallel multi-call assignment: _, _ = f(), g()
					rhsExpr := assign.Rhs[i]
					call, ok := rhsExpr.(*ast.CallExpr)
					if !ok {
						continue
					}
					rhsCall = call
					tv, ok := pass.TypesInfo.Types[rhsExpr]
					if !ok {
						continue
					}
					rhsType = tv.Type
				}

				if rhsType == nil {
					continue
				}

				// Check if the type implements error interface, including via
				// a pointer receiver (e.g. func (e *MyError) Error() string
				// with a function returning MyError by value).
				if !types.Implements(rhsType, errorInterface) && !types.Implements(types.NewPointer(rhsType), errorInterface) {
					continue
				}

				// Get the function/method name for the diagnostic
				funcName := getFuncNameFromCall(rhsCall)

				// Check for Close() method with explanatory comment
				if funcName == "Close" && hasExplanatoryComment(assign, comments, pass.Fset) {
					continue
				}

				message := "discarding error from " + funcName + " via blank assignment; handle or wrap-and-return"
				if funcName == "Close" {
					message = "discarding error from " + funcName + " via blank assignment; handle, wrap-and-return, or add explanatory comment to suppress"
				}
				pass.Report(analysis.Diagnostic{
					Pos:     assign.Pos(),
					Message: message,
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
// It excludes "// want" test annotations and validates that the comment is substantive
func hasExplanatoryComment(assign *ast.AssignStmt, comments []commentInfo, fset *token.FileSet) bool {
	assignLine := fset.Position(assign.Pos()).Line
	for _, c := range comments {
		if c.line == assignLine || c.line == assignLine-1 {
			// Skip "// want" test annotations (analysistest convention: `// want "..."`)
			if strings.Contains(c.text, `// want "`) {
				continue
			}
			// Extract comment content (strip // or /* */ prefix and whitespace)
			content := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(c.text, "//"), "/*"))
			content = strings.TrimSuffix(content, "*/")
			content = strings.TrimSpace(content)
			// Require non-empty content
			if content == "" {
				continue
			}
			// Exclude section separators (lines containing only = or - characters)
			if strings.Trim(content, "=-") == "" {
				continue
			}
			return true
		}
	}
	return false
}

// getFuncNameFromCall extracts the function/method name from a CallExpr.
func getFuncNameFromCall(call *ast.CallExpr) string {
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
