package process

import (
	"testing"
	"time"

	"github.com/feralxr/agent-fleet/internal/config"
)

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

func TestFormatUptime(t *testing.T) {
	if FormatUptime(0) != "" {
		t.Fatal("zero")
	}
	if FormatUptime(5*time.Second) != "5s" {
		t.Fatal(FormatUptime(5 * time.Second))
	}
	if FormatUptime(65*time.Second) != "1m05s" {
		t.Fatal(FormatUptime(65 * time.Second))
	}
}
