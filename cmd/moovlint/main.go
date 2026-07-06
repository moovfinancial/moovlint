package main

import (
	"github.com/moovfinancial/moovlint/analyzers/mockcheck"
	"github.com/moovfinancial/moovlint/analyzers/spanevents"
	"github.com/moovfinancial/moovlint/analyzers/spanrequired"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		spanevents.Analyzer,
		spanrequired.Analyzer,
		mockcheck.Analyzer,
	)
}
