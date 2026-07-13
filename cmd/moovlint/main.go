package main

import (
	"github.com/moovfinancial/moovlint"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(linters.AllAnalyzers()...)
}
