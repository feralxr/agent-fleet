package process

import (
	"testing"

	"github.com/feralxr/agent-fleet/internal/config"
)

func TestLooksLikePrompt(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Allow edit? (y/n)", true},
		{"Do you want to continue?", true},
		{"compiling package...", false},
		{"", false},
	}
	for _, c := range cases {
		if got := looksLikePrompt(c.in); got != c.want {
			t.Errorf("looksLikePrompt(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestStripANSI(t *testing.T) {
	in := "\x1b[31mred\x1b[0m plain"
	got := stripANSI(in)
	if got != "red plain" {
		t.Fatalf("got %q", got)
	}
}

func TestNewManagerListsAgents(t *testing.T) {
	m := NewManager(".", nil)
	list := m.List()
	if len(list) < 4 {
		t.Fatalf("expected >=4 agents, got %d", len(list))
	}
}

func TestSendNotRunning(t *testing.T) {
	m := NewManager(".", nil)
	err := m.Send(config.AgentClaude, "hi")
	if err == nil {
		t.Fatal("expected error")
	}
}
