package linters

import "testing"

func TestPluginSettings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings any
		enabled  string
		wantErr  bool
	}{
		{name: "nil", enabled: "false"},
		{name: "empty", settings: map[string]any{}, enabled: "false"},
		{name: "enabled", settings: map[string]any{
			"fixtureplacement": map[string]any{"enabled": true, "fixture-packages": "/builders$"},
			"modelplacement":   map[string]any{"enabled": true, "model-files": "^types.go$"},
		}, enabled: "true"},
		{name: "unknown", settings: map[string]any{"fixtureplacment": true}, wantErr: true},
		{name: "wrong type", settings: map[string]any{"modelplacement": map[string]any{"enabled": "yes"}}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plugin, err := New(tc.settings)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected settings error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			analyzers, err := plugin.BuildAnalyzers()
			if err != nil {
				t.Fatal(err)
			}
			found := 0
			for _, a := range analyzers {
				if a.Name == "fixtureplacement" || a.Name == "modelplacement" {
					found++
					if got := a.Flags.Lookup("enabled").Value.String(); got != tc.enabled {
						t.Errorf("%s enabled = %s, want %s", a.Name, got, tc.enabled)
					}
					if tc.name == "enabled" {
						flag, want := "fixture-packages", "/builders$"
						if a.Name == "modelplacement" {
							flag, want = "model-files", "^types.go$"
						}
						if got := a.Flags.Lookup(flag).Value.String(); got != want {
							t.Errorf("%s %s = %s, want %s", a.Name, flag, got, want)
						}
					}
				}
			}
			if found != 2 {
				t.Fatalf("got %d placement analyzers, want 2", found)
			}
		})
	}
}

func TestAnalyzerConfigIsolation(t *testing.T) {
	first := AllAnalyzers()
	second := AllAnalyzers()
	for i, a := range first {
		if a.Name != "fixtureplacement" && a.Name != "modelplacement" && a.Name != "mockcheck" {
			continue
		}
		name, value := "enabled", "true"
		if a.Name == "mockcheck" {
			name, value = "allow-interfaces", "^$"
		}
		before := second[i].Flags.Lookup(name).Value.String()
		if err := a.Flags.Set(name, value); err != nil {
			t.Fatal(err)
		}
		if got := second[i].Flags.Lookup(name).Value.String(); got != before {
			t.Errorf("%s shares mutable configuration between instances", a.Name)
		}
	}
}
