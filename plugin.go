package linters

import (
	"github.com/golangci/plugin-module-register/register"
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
	return AllAnalyzers(), nil
}

func (p *Plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
