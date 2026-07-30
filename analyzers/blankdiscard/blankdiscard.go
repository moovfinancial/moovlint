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
	Doc:  "detects blank identifier assignments where the discarded value implements the error interface",
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
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			// Only check simple assignment (=), not declaration (:=)
			if assign.Tok != token.ASSIGN {
				return true
			}

			// Check if any LHS blank identifier corresponds to an error type
			for i, lhs := range assign.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok || id.Name != "_" {
					continue
				}

				// Get the corresponding RHS expression's type
				if !isDiscardingError(pass, assign, i) {
					continue
				}

				// Check if there's an explanatory comment
				if hasExplanatoryComment(pass, file, assign) {
					continue
				}

				pass.Report(analysis.Diagnostic{
					Pos:     assign.Pos(),
					Message: "discarding error return value; handle the error or use an explanatory comment if intentional",
				})
				// Only report once per statement - break out of LHS loop
				break
			}
			return true
		})
	}
	return nil, nil
}

// isDiscardingError checks if the i-th LHS position corresponds to an error type.
func isDiscardingError(pass *analysis.Pass, assign *ast.AssignStmt, lhsIndex int) bool {
	// For simple single assignments like `_ = f()`
	if len(assign.Rhs) == 1 {
		rhs := assign.Rhs[0]
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			return false
		}

		// Get the type of the call expression
		tv, ok := pass.TypesInfo.Types[call]
		if !ok {
			return false
		}

		// Handle tuple returns (multiple values)
		if tuple, ok := tv.Type.(*types.Tuple); ok {
			if lhsIndex >= tuple.Len() {
				return false
			}
			return implementsError(tuple.At(lhsIndex).Type())
		}

		// Single return value
		if lhsIndex == 0 {
			return implementsError(tv.Type)
		}
	}

	return false
}

// implementsError checks if the given type implements the error interface.
func implementsError(t types.Type) bool {
	if t == nil {
		return false
	}

	// Get the error interface from the universe scope
	errorType := types.Universe.Lookup("error").Type()

	// Check if the type is exactly the error interface
	if types.Identical(t, errorType) {
		return true
	}

	// Check if the type implements the error interface
	errorInterface := errorType.Underlying().(*types.Interface)
	return types.Implements(t, errorInterface)
}

// hasExplanatoryComment checks if there's a comment on the same line or the preceding line
// that serves as an explanation for the discarded error.
func hasExplanatoryComment(pass *analysis.Pass, file *ast.File, assign *ast.AssignStmt) bool {
	assignLine := pass.Fset.Position(assign.Pos()).Line

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			commentLine := pass.Fset.Position(c.Pos()).Line

			// Check same line or immediately preceding line
			if commentLine == assignLine || commentLine == assignLine-1 {
				// Ignore analysistest directives (// want "...")
				text := strings.TrimPrefix(c.Text, "//")
				text = strings.TrimSpace(text)
				if strings.HasPrefix(text, "want ") || strings.HasPrefix(text, "want\"") {
					continue
				}
				return true
			}
		}
	}
	return false
}
