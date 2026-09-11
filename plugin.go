package linters

import (
	"fmt"

	"github.com/golangci/plugin-module-register/register"
	"github.com/moovfinancial/moovlint/analyzers/fixtureplacement"
	"github.com/moovfinancial/moovlint/analyzers/mockcheck"
	"github.com/moovfinancial/moovlint/analyzers/modelplacement"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("moovlint", New)
}

type Settings struct {
	MockCheck        mockcheck.Config        `json:"mockcheck"`
	FixturePlacement fixtureplacement.Config `json:"fixtureplacement"`
	ModelPlacement   modelplacement.Config   `json:"modelplacement"`
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
