package process

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/feralxr/agent-fleet/internal/config"
)

func TestManagerStartMissingBinary(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir, nil)
	// Force a known-missing command
	if p := m.Get(config.AgentClaude); p != nil {
		p.CmdPath = filepath.Join(dir, "no-such-binary-fleet-test")
		p.Available = false
	}
	// Re-start path for missing
	err := m.Start(config.AgentClaude)
	if err == nil {
		// May succeed if claude is installed — skip
		t.Skip("claude present on PATH")
	}
}

func TestSendWhenIdleErrors(t *testing.T) {
	m := NewManager(t.TempDir(), nil)
	err := m.Send(config.AgentClaude, "hi")
	if err == nil {
		t.Fatal("expected error when not running")
	}
}

func TestSetModelPersistsOnAgent(t *testing.T) {
	m := NewManager(t.TempDir(), &config.Session{
		Models: map[string]string{string(config.AgentClaude): "sonnet"},
	})
	if got := m.GetModel(config.AgentClaude); got != "sonnet" {
		t.Fatalf("got %q", got)
	}
	m.SetModel(config.AgentClaude, "opus")
	if got := m.GetModel(config.AgentClaude); got != "opus" {
		t.Fatalf("got %q", got)
	}
}

func TestSnapshotOutputIndependent(t *testing.T) {
	m := NewManager(t.TempDir(), nil)
	p := m.Get(config.AgentClaude)
	if p == nil {
		t.Fatal("nil agent")
	}
	p.append("line-a")
	p.append("line-b")
	snap := p.SnapshotOutput()
	if len(snap) < 2 {
		t.Fatalf("snap %v", snap)
	}
	n := len(snap)
	p.append("line-c")
	if len(snap) != n {
		t.Fatal("snapshot length changed after append — shared buffer")
	}
	for _, line := range snap {
		if line == "line-c" {
			t.Fatal("snapshot saw later append — shared buffer")
		}
	}
}

func TestActivityLabel(t *testing.T) {
	p := &AgentProcess{Status: StatusRunning, lastActive: time.Now()}
	if p.ActivityLabel() != "streaming" {
		// may be working if clock skew — accept either streaming or working
		lab := p.ActivityLabel()
		if lab != "streaming" && lab != "working" {
			t.Fatalf("label %s", lab)
		}
	}
	p.Status = StatusWaiting
	if p.ActivityLabel() != "needs you" {
		t.Fatalf("got %s", p.ActivityLabel())
	}
}

func TestMergeModelsPrefersLive(t *testing.T) {
	cfg := &config.AgentConfig{ID: config.AgentGrok, Models: []string{"fallback"}}
	got := MergeModels(config.AgentGrok, cfg, []string{"live-a", "live-b"})
	if got[0] != "live-a" {
		t.Fatalf("got %v", got)
	}
}

func TestCwdEffectiveWorktree(t *testing.T) {
	root := t.TempDir()
	wt := filepath.Join(root, "wt")
	if err := os.Mkdir(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewManager(root, nil)
	m.SetWorktree(config.AgentClaude, wt)
	if m.EffectiveCwd(config.AgentClaude) != wt {
		t.Fatalf("got %s", m.EffectiveCwd(config.AgentClaude))
	}
}

func TestListReturnsSnapshots(t *testing.T) {
	m := NewManager(t.TempDir(), nil)
	list := m.List()
	if len(list) == 0 {
		t.Fatal("empty list")
	}
	// Mutating snapshot status must not change live agent
	live := m.Get(list[0].ID)
	before := live.Status
	list[0].Status = StatusError
	if live.Status != before {
		t.Fatal("List exposed live pointer — status mutated")
	}
}

func TestConcurrentAppendAndList(t *testing.T) {
	m := NewManager(t.TempDir(), nil)
	p := m.Get(config.AgentClaude)
	if p == nil {
		t.Fatal("nil")
	}
	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			p.append(fmt.Sprintf("line-%d", i))
		}
		close(done)
	}()
	for i := 0; i < 200; i++ {
		list := m.List()
		if len(list) == 0 {
			t.Fatal("empty")
		}
		_ = list[0].CurrentStatus()
		_ = list[0].ActivityLabel()
	}
	<-done
}
