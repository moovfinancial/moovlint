package enumliteral

import "testdata/enumliteral/status"

func testOnly(s status.Status) bool {
	return s == "created"
}
