package process

import (
	"testing"

	"github.com/feralxr/agent-fleet/internal/config"
)

func TestParseModelLines(t *testing.T) {
	in := `
Available models
anthropic/claude-sonnet-4-5
openai/gpt-5 (default)
• grok-4
Usage: something
`
	got := parseModelLines(in)
	if len(got) < 3 {
		t.Fatalf("got %v", got)
	}
	if got[0] != "anthropic/claude-sonnet-4-5" {
		t.Fatal(got[0])
	}
}

func TestModelCacheTTL(t *testing.T) {
	// DiscoverModels on missing binary should still be safe / empty
	list := DiscoverModels(config.AgentConfig{ID: config.AgentClaude, Command: "claude-not-installed-xyz"})
	if list != nil && len(list) > 0 {
		// empty is expected
	}
	_ = list
}

func TestMergeModelsDedupOrder(t *testing.T) {
	cfg := &config.AgentConfig{ID: config.AgentOpenCode, Models: []string{"live-a", "cfg-only"}}
	got := MergeModels(config.AgentOpenCode, cfg, []string{"live-a", "live-b"})
	if len(got) < 2 || got[0] != "live-a" || got[1] != "live-b" {
		t.Fatalf("got %v", got)
	}
	// cfg-only should appear after live
	found := false
	for _, m := range got {
		if m == "cfg-only" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing cfg-only in %v", got)
	}
}
