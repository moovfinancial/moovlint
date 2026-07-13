package structtags

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/moovfinancial/moovlint/repocheck"
)

type StructTagsChecker struct{}

func (StructTagsChecker) Name() string { return "structtags" }

func (StructTagsChecker) Check(root string) ([]repocheck.Diagnostic, error) {
	files, err := repocheck.FindGoFiles(root)
	if err != nil {
		return nil, err
	}

	var diags []repocheck.Diagnostic
	for _, path := range files {
		d := checkStructTags(path)
		diags = append(diags, d...)
	}
	return diags, nil
}

func checkStructTags(path string) []repocheck.Diagnostic {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var diags []repocheck.Diagnostic
	ast.Inspect(file, func(n ast.Node) bool {
		structType, ok := n.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range structType.Fields.List {
			if field.Tag == nil {
				continue
			}
			tag := field.Tag.Value
			jsonName := extractTagValue(tag, "json")
			if jsonName == "" || jsonName == "-" {
				continue
			}
			jsonName = strings.Split(jsonName, ",")[0]

			if !isCamelCase(jsonName) {
				diags = append(diags, repocheck.Diagnostic{
					Path:    path,
					Line:    fset.Position(field.Pos()).Line,
					Message: fmt.Sprintf("json tag %q should be camelCase", jsonName),
				})
			}

			if strings.HasSuffix(jsonName, "ID") || strings.HasSuffix(jsonName, "Id") {
				diags = append(diags, repocheck.Diagnostic{
					Path:    path,
					Line:    fset.Position(field.Pos()).Line,
					Message: fmt.Sprintf("json tag %q should use lowercase 'Id' not 'ID' (e.g. 'accountId' not 'accountID')", jsonName),
				})
			}

			if hasFieldNames(field) {
				name := field.Names[0].Name
				if strings.HasSuffix(name, "On") || strings.HasSuffix(name, "At") {
					if !strings.HasSuffix(jsonName, "On") && !strings.HasSuffix(jsonName, "At") {
						diags = append(diags, repocheck.Diagnostic{
							Path:    path,
							Line:    fset.Position(field.Pos()).Line,
							Message: fmt.Sprintf("timestamp field %s has json tag %q; expected suffix 'On' or 'At'", name, jsonName),
						})
					}
				}
			}
		}
		return true
	})
	return diags
}

func extractTagValue(tag, key string) string {
	tag = strings.Trim(tag, "`")
	parts := strings.Split(tag, " ")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, key+":") {
			val := strings.TrimPrefix(p, key+":")
			val = strings.Trim(val, `"`)
			return val
		}
	}
	return ""
}

func isCamelCase(s string) bool {
	if s == "" {
		return true
	}
	if s[0] >= 'A' && s[0] <= 'Z' {
		return false
	}
	if strings.Contains(s, "_") {
		return false
	}
	return true
}

func hasFieldNames(field *ast.Field) bool {
	return len(field.Names) > 0
}
