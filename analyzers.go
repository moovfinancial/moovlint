package linters

import (
	"github.com/moovfinancial/moovlint/analyzers/blankdiscard"
	"github.com/moovfinancial/moovlint/analyzers/contextcancel"
	"github.com/moovfinancial/moovlint/analyzers/controllerassert"
	"github.com/moovfinancial/moovlint/analyzers/ctornilguard"
	"github.com/moovfinancial/moovlint/analyzers/enumcast"
	"github.com/moovfinancial/moovlint/analyzers/grpcserver"
	"github.com/moovfinancial/moovlint/analyzers/grpcstatus"
	"github.com/moovfinancial/moovlint/analyzers/httpdecodeflag"
	"github.com/moovfinancial/moovlint/analyzers/logformat"
	"github.com/moovfinancial/moovlint/analyzers/mapderef"
	"github.com/moovfinancial/moovlint/analyzers/midusage"
	"github.com/moovfinancial/moovlint/analyzers/mockcheck"
	"github.com/moovfinancial/moovlint/analyzers/moneyfloat"
	"github.com/moovfinancial/moovlint/analyzers/nolintguard"
	"github.com/moovfinancial/moovlint/analyzers/oteltags"
	"github.com/moovfinancial/moovlint/analyzers/repoerrorflags"
	"github.com/moovfinancial/moovlint/analyzers/requiregoroutine"
	"github.com/moovfinancial/moovlint/analyzers/spancontext"
	"github.com/moovfinancial/moovlint/analyzers/spanerrors"
	"github.com/moovfinancial/moovlint/analyzers/spanevents"
	"github.com/moovfinancial/moovlint/analyzers/spanlifecycle"
	"github.com/moovfinancial/moovlint/analyzers/spanname"
	"github.com/moovfinancial/moovlint/analyzers/spanrequired"
	"github.com/moovfinancial/moovlint/analyzers/subtestassert"
	"github.com/moovfinancial/moovlint/analyzers/testsleep"
	"github.com/moovfinancial/moovlint/analyzers/timeinject"
	"github.com/moovfinancial/moovlint/analyzers/uuidgen"
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
		repoerrorflags.Analyzer,
		timeinject.Analyzer,
		contextcancel.Analyzer,
		nolintguard.Analyzer,
		blankdiscard.Analyzer,
		uuidgen.Analyzer,
		requiregoroutine.Analyzer,
		spanname.Analyzer,
		testsleep.Analyzer,
		logformat.Analyzer,
		moneyfloat.Analyzer,
		spanerrors.Analyzer,
		mapderef.Analyzer,
		subtestassert.Analyzer,
		ctornilguard.Analyzer,
		enumcast.Analyzer,
	}
}
