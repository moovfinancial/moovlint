// Package testifysubtestassertion flags testify suite Assert/Require calls made inside
// a t.Run subtest closure. Calling suite-level assertions from inside a subtest closure
// misattributes failures to the outer test method; assertions inside a subtest closure
// must be bound to a local assert.New(t) / require.New(t) scoped to the inner *testing.T.
//
// Pattern caught:
//   func (s *MySuite) TestThing(t *testing.T) {
//       s.Run("nested", func(t *testing.T) {
//           s.Assert.Equal(t, 1, 1)       // want: bind to assert.New(t).Equal(...)
//           s.Require.NoError(t, nil)     // want: bind to require.New(t).NoError(...)
//       })
//   }
//
// Restricts scanning to *_test.go files — production callers cannot reach s.Run.
package testifysubtestassertion

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "testifysubtestassertion",
	Doc:  "flags testify suite Assert/Require calls inside t.Run / s.Run subtest closures — bind assertions to assert.New(t) / require.New(t) scoped to the inner *testing.T instead.",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if !strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			ast.Inspect(fn.Body, func(m ast.Node) bool {
				call, ok := m.(*ast.CallExpr)
				if !ok || !isSubtestRunCall(call) {
					return true
				}
				if len(call.Args) < 2 {
					return true
				}
				closure, ok := call.Args[1].(*ast.FuncLit)
				if !ok {
					return true
				}
				ast.Inspect(closure.Body, func(c ast.Node) bool {
					if reportSuiteAssertion(pass, c) {
						return false
					}
					return true
				})
				return true
			})
			return true
		})
	}
	return nil, nil
}

// isSubtestRunCall matches `something.Run(name, closure)` call shapes:
//   - t.Run, timeRun.Run, etc.: (*ast.Ident).Run
//   - s.T().Run, anything.T().Run: (*ast.CallExpr wrapping (*ast.SelectorExpr with Sel="T")).Run
func isSubtestRunCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Run" {
		return false
	}
	switch v := sel.X.(type) {
	case *ast.Ident:
		return true
	case *ast.CallExpr:
		inner, ok := v.Fun.(*ast.SelectorExpr)
		return ok && inner.Sel.Name == "T"
	}
	return false
}

// reportSuiteAssertion inspects call.Fun's selector chain for an ".Assert" or
// ".Require" segment above the leaf selector and emits a diagnostic at the call's
// position. Returns true when the call is reported so the inspector can prune.
func reportSuiteAssertion(pass *analysis.Pass, n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	head := call.Fun
	for head != nil {
		switch v := head.(type) {
		case *ast.SelectorExpr:
			if v.Sel.Name == "Assert" || v.Sel.Name == "Require" {
				pass.Report(analysis.Diagnostic{
					Pos: call.Pos(),
					Message: fmt.Sprintf(
						"suite %q called inside t.Run subtest closure — bind assertions to assert.New(t) / require.New(t) scoped to the inner *testing.T instead",
						v.Sel.Name,
					),
				})
				return true
			}
			head = v.X
		default:
			return false
		}
	}
	return false
}
