package errswallow

// fetchInt returns no error — `_ = fetchInt()` MUST NOT be flagged.
func fetchInt() int { return 1 }

// fetchWithErr returns an error — `_ = fetchWithErr()` MUST be flagged.
func fetchWithErr() (string, error) { return "", nil }

func uses() {
	_ = fetchInt() // ok: no error return

	_ = fetchWithErr() // want "error from call is discarded"

	// Short variable declarations are *ast.GenDecl, not AssignStmt, so the
	// analyzer does not flag them. Documented limitation; most call sites
	// use the AssignStmt form when discarding.
	_, _ := fetchWithErr()
}
