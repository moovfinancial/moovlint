package blankdiscard

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/moovfinancial/moovlint/internal/moovutil"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "blankdiscard",
	Doc:  "detects blank discards (_ =) of error and sql.Result returns without an inline justification comment",
	Run:  run,
}

var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func run(pass *analysis.Pass) (any, error) {
	if !moovutil.IsServicePackage(pass.Pkg.Path()) {
		return nil, nil
	}

	for _, file := range pass.Files {
		commentLines := make(map[int]bool)
		for _, cg := range file.Comments {
			for _, c := range cg.List {
				if isJustificationComment(c.Text) {
					commentLines[pass.Fset.Position(c.Pos()).Line] = true
				}
			}
		}

		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			checkAssign(pass, assign, commentLines)
			return true
		})
	}
	return nil, nil
}

func checkAssign(pass *analysis.Pass, assign *ast.AssignStmt, commentLines map[int]bool) {
	if commentLines[pass.Fset.Position(assign.Pos()).Line] {
		return
	}
	resultType := types.Type(nil)
	for i, lhs := range assign.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || id.Name != "_" {
			continue
		}
		var t types.Type
		if len(assign.Rhs) == 1 && len(assign.Lhs) > 1 {
			if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
				if tv, ok := pass.TypesInfo.Types[call]; ok {
					if tuple, ok := tv.Type.(*types.Tuple); ok && i < tuple.Len() {
						t = tuple.At(i).Type()
					}
				}
			}
		} else if i < len(assign.Rhs) {
			t = pass.TypesInfo.TypeOf(assign.Rhs[i])
		}
		if t == nil {
			continue
		}
		if isErrorType(t) {
			pass.Report(analysis.Diagnostic{
				Pos:     assign.Pos(),
				Message: "error return blank-discarded without justification; handle it or add an inline comment explaining why it is safe to ignore",
			})
			return
		}
		if resultType == nil && isSQLResult(t) {
			resultType = t
		}
	}
	if resultType != nil {
		pass.Report(analysis.Diagnostic{
			Pos:     assign.Pos(),
			Message: "sql.Result blank-discarded without justification; check RowsAffected or add an inline comment explaining why it is safe to ignore",
		})
	}
}

func isJustificationComment(text string) bool {
	t := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	return t != "" && !strings.HasPrefix(t, "want")
}

func isErrorType(t types.Type) bool {
	return types.Implements(t, errorIface)
}

func isSQLResult(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil &&
		obj.Pkg().Path() == "database/sql" && obj.Name() == "Result"
}
