package nolintguard

import "fmt"

func badBare() {
	_ = fmt.Sprintf("test") //nolint // want "bare //nolint without a linter name"
}

func badAll() {
	_ = fmt.Sprintf("test") //nolint:all // want "//nolint:all is not allowed"
}

func badNoExplanation() {
	_ = fmt.Sprintf("test") //nolint:errcheck // want "//nolint directive requires an explanation"
}

func goodSpecific() {
	_ = fmt.Sprintf("test") //nolint:errcheck // intentional: test helper
}
