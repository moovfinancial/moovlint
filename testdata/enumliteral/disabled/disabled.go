package disabled

import "testdata/enumliteral/status"

func compare(s status.Status) bool {
	return s == "created"
}
