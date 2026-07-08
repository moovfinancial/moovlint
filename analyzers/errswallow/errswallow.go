// Package errswallow implements an analyzer that detects discarded errors.
//
// It flags assignments of the form `_ = <call>` where the called function
// returns an error, indicating the error is silently swallowed.
//
// errcheck also reports this, but only when its configuration enables it.
// errswallow has the same heuristic regardless of repo-level config so the
// rule is consistent across every repo that registers moovlint.
package errswallow

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// errorInterface is the built-in `error` type.
var errorInterface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

var Analyzer = &analysis.Analyzer{
	Name: "errswallow",
	Doc:  "detects _ = <call> assignments where the call returns an error (discarded error)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			// Skip declarations like `_ = x` with no RHS call to inspect.
			if len(assign.Rhs) != 1 {
				return true
			}

			call, ok := unwrapCall(assign.Rhs[0])
			if !ok {
				return true
			}

			tv, ok := pass.TypesInfo.Types[call.Fun]
			if !ok {
				return true
			}
			sig, ok := tv.Type.(*types.Signature)
			if !ok || sig.Results().Len() == 0 {
				return true
			}

			// Determine which return positions are error-typed.
			results := sig.Results()
			errPositions := make([]bool, results.Len())
			anyErr := false
			for i := 0; i < results.Len(); i++ {
				errPositions[i] = isErrorType(results.At(i).Type())
				anyErr = anyErr || errPositions[i]
			}
			if !anyErr {
				return true
			}

			// Walk the LHS slice. Each `_` paired with an error-typed
			// return position gets a diagnostic on that underscore.
			for i, lhs := range assign.Lhs {
				if i >= len(errPositions) || !errPositions[i] {
					continue
				}
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != "_" {
					continue
				}
				pass.Report(analysis.Diagnostic{
					Pos:     ident.Pos(),
					Message: "error from call is discarded with `_`; name the return value, log the error, or return it",
				})
			}
			return true
		})
	}
	return nil, nil
}

func unwrapCall(expr ast.Expr) (*ast.CallExpr, bool) {
	if call, ok := expr.(*ast.CallExpr); ok {
		return call, true
	}
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		if call, ok := unary.X.(*ast.CallExpr); ok {
			return call, true
		}
	}
	return nil, false
}

func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	if types.Implements(t, errorInterface) {
		return true
	}
	// Named types whose underlying is `error` itself (rare but legal).
	if named, ok := t.(*types.Named); ok {
		if named.Obj() != nil && named.Obj().Name() == "error" {
			return true
		}
	}
	return false
}
