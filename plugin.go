package linters

import (
	"fmt"

	"github.com/golangci/plugin-module-register/register"
	"github.com/moovfinancial/moovlint/analyzers/ctornilguard"
	"github.com/moovfinancial/moovlint/analyzers/enumcast"
	"github.com/moovfinancial/moovlint/analyzers/enumliteral"
	"github.com/moovfinancial/moovlint/analyzers/fixtureplacement"
	"github.com/moovfinancial/moovlint/analyzers/mapderef"
	"github.com/moovfinancial/moovlint/analyzers/midusage"
	"github.com/moovfinancial/moovlint/analyzers/mockcheck"
	"github.com/moovfinancial/moovlint/analyzers/modelplacement"
	"github.com/moovfinancial/moovlint/analyzers/repoerrorflags"
	"github.com/moovfinancial/moovlint/analyzers/spanerrors"
	"github.com/moovfinancial/moovlint/analyzers/spannersql"
	"github.com/moovfinancial/moovlint/analyzers/spannertxcapture"
	"github.com/moovfinancial/moovlint/analyzers/subtestassert"
	"github.com/moovfinancial/moovlint/analyzers/testlog"
	"github.com/moovfinancial/moovlint/analyzers/testsleep"
	"github.com/moovfinancial/moovlint/analyzers/timeinject"
	"github.com/moovfinancial/moovlint/analyzers/wrapnil"
	"github.com/moovfinancial/moovlint/analyzers/writegate"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("moovlint", New)
}

type Settings struct {
	MockCheck        mockcheck.Config        `json:"mockcheck"`
	FixturePlacement fixtureplacement.Config `json:"fixtureplacement"`
	ModelPlacement   modelplacement.Config   `json:"modelplacement"`
	Ctornilguard     ctornilguard.Config     `json:"ctornilguard"`
	SpanErrors       spanerrors.Config       `json:"spanerrors"`
	EnumCast         enumcast.Config         `json:"enumcast"`
	RepoErrorFlags   repoerrorflags.Config   `json:"repoerrorflags"`
	TestSleep        testsleep.Config        `json:"testsleep"`
	MapDeref         mapderef.Config         `json:"mapderef"`
	Writegate        writegate.Config        `json:"writegate"`
	TestLog          testlog.Config          `json:"testlog"`
	SpannerTxCapture spannertxcapture.Config `json:"spannertxcapture"`
	SpannerSQL       spannersql.Config       `json:"spannersql"`
	WrapNil          wrapnil.Config          `json:"wrapnil"`
	EnumLiteral      enumliteral.Config      `json:"enumliteral"`
	TimeInject       timeinject.Config       `json:"timeinject"`
	SubtestAssert    subtestassert.Config    `json:"subtestassert"`
	MidUsage         midusage.Config         `json:"midusage"`
}

type Plugin struct {
	settings Settings
}

func New(settings any) (register.LinterPlugin, error) {
	cfg, err := register.DecodeSettings[Settings](settings)
	if err != nil {
		return nil, fmt.Errorf("moovlint settings: %w", err)
	}
	return &Plugin{settings: cfg}, nil
}

func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return configuredAnalyzers(p.settings), nil
}

func (p *Plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
