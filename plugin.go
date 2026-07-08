package linters

import (
	"github.com/golangci/plugin-module-register/register"
	"github.com/moovfinancial/moovlint/analyzers/errswallow"
	"github.com/moovfinancial/moovlint/analyzers/mockcheck"
	"github.com/moovfinancial/moovlint/analyzers/spanevents"
	"github.com/moovfinancial/moovlint/analyzers/spanrequired"
	"github.com/moovfinancial/moovlint/analyzers/thelper"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("moovlint", New)
}

type Plugin struct{}

func New(settings any) (register.LinterPlugin, error) {
	return &Plugin{}, nil
}

func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{
		spanevents.Analyzer,
		spanrequired.Analyzer,
		mockcheck.Analyzer,
		errswallow.Analyzer,
		thelper.Analyzer,
	}, nil
}

func (p *Plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
