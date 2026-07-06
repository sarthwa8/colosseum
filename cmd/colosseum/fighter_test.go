package main

import (
	"strings"
	"testing"
)

func TestParseFighterGemini(t *testing.T) {
	cfg, err := parseFighter("gemini:gemini-3.5-flash", "A", nil)
	if err != nil {
		t.Fatalf("parseFighter: %v", err)
	}
	if cfg.Provider.Name() != "gemini" {
		t.Errorf("provider = %q, want gemini", cfg.Provider.Name())
	}
	if cfg.Model != "gemini-3.5-flash" {
		t.Errorf("model = %q", cfg.Model)
	}
}

func TestParseFighterUnknownProviderListsGemini(t *testing.T) {
	_, err := parseFighter("nope:model", "A", nil)
	if err == nil || !strings.Contains(err.Error(), "gemini") {
		t.Errorf("err = %v, want provider list mentioning gemini", err)
	}
}

func TestFighterFactoryGemini(t *testing.T) {
	f, err := fighterFactory("gemini:gemini-3.1-pro")
	if err != nil {
		t.Fatalf("fighterFactory: %v", err)
	}
	if f.Label != "gemini-3.1-pro" {
		t.Errorf("label = %q", f.Label)
	}
	if got := f.New(nil).Name(); got != "gemini" {
		t.Errorf("provider = %q, want gemini", got)
	}
}
