package protobuf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/moovfinancial/moovlint/repocheck"
)

type ProtobufChecker struct{}

func (ProtobufChecker) Name() string { return "protobuf" }

func (ProtobufChecker) Check(root string) ([]repocheck.Diagnostic, error) {
	files, err := findProtoFiles(root)
	if err != nil {
		return nil, err
	}

	var diags []repocheck.Diagnostic
	for _, path := range files {
		d := checkProtoFile(path)
		diags = append(diags, d...)
	}
	return diags, nil
}

func findProtoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "third_party" || name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".proto") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

var (
	fieldRe    = regexp.MustCompile(`^\s*(repeated\s+|optional\s+)?(\w+)\s+(\w+)\s*=\s*(\d+)\s*;`)
	reservedRe = regexp.MustCompile(`^\s*reserved\s+(\d+)\s*(?:to\s+(\d+))?\s*;`)
	messageRe  = regexp.MustCompile(`^\s*message\s+(\w+)\s*\{`)
)

type protoField struct {
	name   string
	number int
	line   int
}

type messageScope struct {
	name            string
	startLine       int
	endLine         int
	fields          []protoField
	reservedNumbers map[int]bool
	commentLines    map[int]bool
}

func checkProtoFile(path string) []repocheck.Diagnostic {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")

	scopes := parseMessages(lines)
	if len(scopes) == 0 {
		return nil
	}

	var diags []repocheck.Diagnostic

	for _, scope := range scopes {
		sort.Slice(scope.fields, func(i, j int) bool {
			return scope.fields[i].number < scope.fields[j].number
		})

		usedNumbers := make(map[int]bool)
		for _, f := range scope.fields {
			if usedNumbers[f.number] {
				diags = append(diags, repocheck.Diagnostic{
					Path:    path,
					Line:    f.line,
					Message: fmt.Sprintf("protobuf field %s reuses field number %d; field numbers are permanent and must be unique", f.name, f.number),
				})
			}
			usedNumbers[f.number] = true
		}

		for i := 1; i < len(scope.fields); i++ {
			if scope.fields[i].number == scope.fields[i-1].number || scope.fields[i].number == scope.fields[i-1].number+1 {
				continue
			}
			gapStart := scope.fields[i-1].number + 1
			gapEnd := scope.fields[i].number - 1
			if gapEnd < gapStart {
				continue
			}

			allReserved := true
			for n := gapStart; n <= gapEnd; n++ {
				if !scope.reservedNumbers[n] {
					allReserved = false
					break
				}
			}
			if allReserved {
				continue
			}

			hasComment := false
			for ln := scope.fields[i-1].line; ln < scope.fields[i].line-1; ln++ {
				if scope.commentLines[ln] {
					hasComment = true
					break
				}
			}
			if hasComment {
				continue
			}

			diags = append(diags, repocheck.Diagnostic{
				Path:    path,
				Line:    scope.fields[i].line,
				Message: fmt.Sprintf("protobuf field %s uses number %d but %d-%d are unused and not reserved or commented; gaps in numbering need an explanation (reserved or comment)", scope.fields[i].name, scope.fields[i].number, gapStart, gapEnd),
			})
		}
	}

	return diags
}

func parseMessages(lines []string) []*messageScope {
	var scopes []*messageScope
	var current *messageScope
	depth := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if current == nil {
			if m := messageRe.FindStringSubmatch(trimmed); m != nil {
				current = &messageScope{
					name:            m[1],
					startLine:       i,
					reservedNumbers: make(map[int]bool),
					commentLines:    make(map[int]bool),
				}
				depth = 1
			}
			continue
		}

		depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

		if depth <= 0 {
			current.endLine = i
			scopes = append(scopes, current)
			current = nil
			continue
		}

		if strings.HasPrefix(trimmed, "//") {
			current.commentLines[i] = true
			continue
		}

		if m := reservedRe.FindStringSubmatch(trimmed); m != nil {
			start, _ := strconv.Atoi(m[1])
			if m[2] != "" {
				end, _ := strconv.Atoi(m[2])
				for n := start; n <= end; n++ {
					current.reservedNumbers[n] = true
				}
			} else {
				current.reservedNumbers[start] = true
			}
			continue
		}

		if m := fieldRe.FindStringSubmatch(trimmed); m != nil {
			num, _ := strconv.Atoi(m[4])
			current.fields = append(current.fields, protoField{
				name:   m[3],
				number: num,
				line:   i + 1,
			})
		}

		if m := messageRe.FindStringSubmatch(trimmed); m != nil {
			inner := &messageScope{
				name:            m[1],
				startLine:       i,
				reservedNumbers: make(map[int]bool),
				commentLines:    make(map[int]bool),
			}
			scopes = append(scopes, inner)
			current = inner
			depth = 1
		}
	}

	if current != nil {
		scopes = append(scopes, current)
	}

	return scopes
}
