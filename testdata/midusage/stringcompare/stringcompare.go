package stringcompare

import "github.com/moovfinancial/go-libs/mid"

func BadBothString(a, b mid.ID[mid.Account]) bool {
	return a.String() == b.String() // want "compare mid.ID values with Equals, not through String\\(\\)"
}

func BadPointerString(a *mid.ID[mid.Account], b mid.ID[mid.Account]) bool {
	return a.String() != b.String() // want "compare mid.ID values with Equals"
}

func BadEmpty(a mid.ID[mid.Account]) bool {
	return a.String() == "" // want "use IsEmpty\\(\\) on the mid.ID"
}

func BadEmptyReversed(a mid.ID[mid.Account]) bool {
	return "" != a.String() // want "use IsEmpty\\(\\) on the mid.ID"
}

func OKRawString(a mid.ID[mid.Account], raw string) bool {
	return a.String() == raw
}

func OKNonEmptyLiteral(a mid.ID[mid.Account]) bool {
	return a.String() == "x"
}
