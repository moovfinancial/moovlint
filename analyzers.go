package linters

import (
	"github.com/moovfinancial/moovlint/analyzers/controllerassert"
	"github.com/moovfinancial/moovlint/analyzers/grpcserver"
	"github.com/moovfinancial/moovlint/analyzers/grpcstatus"
	"github.com/moovfinancial/moovlint/analyzers/httpdecodeflag"
	"github.com/moovfinancial/moovlint/analyzers/midusage"
	"github.com/moovfinancial/moovlint/analyzers/mockcheck"
	"github.com/moovfinancial/moovlint/analyzers/oteltags"
	"github.com/moovfinancial/moovlint/analyzers/spancontext"
	"github.com/moovfinancial/moovlint/analyzers/spanevents"
	"github.com/moovfinancial/moovlint/analyzers/spanlifecycle"
	"github.com/moovfinancial/moovlint/analyzers/spanrequired"
	"github.com/moovfinancial/moovlint/analyzers/validationflag"
	"golang.org/x/tools/go/analysis"
)

// AllAnalyzers returns every moovlint analyzer. This is the single source of
// truth consumed by both the golangci-lint plugin and the standalone CLI.
func AllAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		spanevents.Analyzer,
		spanrequired.Analyzer,
		spanlifecycle.Analyzer,
		spancontext.Analyzer,
		mockcheck.Analyzer,
		validationflag.Analyzer,
		grpcstatus.Analyzer,
		grpcserver.Analyzer,
		httpdecodeflag.Analyzer,
		midusage.Analyzer,
		oteltags.Analyzer,
		controllerassert.Analyzer,
	}
}
