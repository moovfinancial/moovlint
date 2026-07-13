package main

import (
	"fmt"
	"os"

	"github.com/moovfinancial/moovlint"
	"github.com/moovfinancial/moovlint/repocheck"
	"github.com/moovfinancial/moovlint/repocheck/migrations"
	"github.com/moovfinancial/moovlint/repocheck/protobuf"
	"github.com/moovfinancial/moovlint/repocheck/structtags"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "repo" {
		runRepoChecks()
		return
	}
	multichecker.Main(linters.AllAnalyzers()...)
}

func runRepoChecks() {
	root := "."
	if len(os.Args) > 2 {
		root = os.Args[2]
	}
	checkers := []repocheck.Checker{
		migrations.MigrationsChecker{},
		structtags.StructTagsChecker{},
		protobuf.ProtobufChecker{},
	}
	diags, err := repocheck.RunAll(root, checkers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	repocheck.PrintDiagnostics(diags)
	if len(diags) > 0 {
		os.Exit(1)
	}
}
