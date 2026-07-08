package thelper

import "testing"

// goodHelperFatal calls t.Helper() before t.Fatal — MUST NOT be flagged.
func goodHelperFatal(t *testing.T, id string) {
	t.Helper()
	t.Fatal("oops: " + id)
}

// badHelperFatal calls t.Fatal without t.Helper() — MUST be flagged.
func badHelperFatal(t *testing.T, id string) { // want "test helper 'badHelperFatal'"
	t.Fatal("oops: " + id)
}

// goodHelperNoFatal does not call t.Fatal/t.Error — MUST NOT be flagged,
// even when it omits t.Helper().
func goodHelperNoFatal(t *testing.T, id string) {
	_ = id
}

// notAHelperDoesNotTakeTestingT has a non-testing.T first param — MUST NOT
// be flagged. The signature gate keeps analyzer output tight.
func notAHelperDoesNotTakeTestingT(s string, id string) {
	_, _ = s, id
}
