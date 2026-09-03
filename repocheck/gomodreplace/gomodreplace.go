package gomodreplace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/moovfinancial/moovlint/repocheck"
)

type GoModReplaceChecker struct{}

func (GoModReplaceChecker) Name() string { return "gomodreplace" }

var ticketRe = regexp.MustCompile(`\b[A-Z][A-Z0-9]*-[0-9]+\b`)

func (GoModReplaceChecker) Check(root string) ([]repocheck.Diagnostic, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var diags []repocheck.Diagnostic
	inBlock := false
	blockJustified := false
	prevCommentTicket := false

	for i, raw := range strings.Split(string(data), "\n") {
		lineNo := i + 1
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if strings.HasPrefix(line, "//") {
				if hasTicketComment(line) {
					blockJustified = true
				}
				continue
			}
			if !hasTicketComment(raw) && !blockJustified {
				diags = append(diags, diag(lineNo, line))
			}
			continue
		}

		if strings.HasPrefix(line, "//") {
			prevCommentTicket = hasTicketComment(line)
			continue
		}

		if !strings.HasPrefix(line, "replace ") {
			prevCommentTicket = false
			continue
		}

		body := strings.TrimSpace(strings.TrimPrefix(line, "replace "))
		if strings.HasSuffix(body, "(") {
			inBlock = true
			blockJustified = hasTicketComment(raw) || prevCommentTicket
			continue
		}
		if !hasTicketComment(raw) && !prevCommentTicket {
			diags = append(diags, diag(lineNo, line))
		}
		prevCommentTicket = false
	}
	return diags, nil
}

func diag(lineNo int, line string) repocheck.Diagnostic {
	rule := strings.SplitN(line, "//", 2)[0]
	return repocheck.Diagnostic{
		Path: "go.mod",
		Line: lineNo,
		Message: fmt.Sprintf("replace directive %q has no tracking ticket comment; replace directives must be temporary and tracked (e.g. // TODO(LINEAR-123))",
			strings.TrimSpace(rule)),
	}
}

func hasTicketComment(line string) bool {
	idx := strings.Index(line, "//")
	if idx < 0 {
		return false
	}
	comment := line[idx:]
	return ticketRe.MatchString(comment) || strings.Contains(strings.ToUpper(comment), "TODO")
}
