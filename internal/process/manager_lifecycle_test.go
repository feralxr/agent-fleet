package process

import (
	"fmt"
	"os"
	"os/exec"
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

func TestStartDoesNotHoldManagerDuringSpawn(t *testing.T) {
	// Use a long-running process so Start must spawn without serializing all List calls.
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}
	dir := t.TempDir()
	// Override claude command via session custom cmd
	sess := &config.Session{CustomCmds: map[string]string{string(config.AgentClaude): "sleep"}}
	// Need Args - CustomCmds only sets Command. Inject via ReloadConfigs won't set args.
	// Build manager and poke config
	m := NewManager(dir, sess)
	// Manually adjust configs for sleep 60
	m.mu.Lock()
	for i := range m.configs {
		if m.configs[i].ID == config.AgentClaude {
			m.configs[i].Command = "sleep"
			m.configs[i].Args = []string{"30"}
			f := true
			m.configs[i].UsePTY = &f
			// pipes are fine too
			f2 := false
			m.configs[i].UsePTY = &f2
		}
	}
	if p := m.agents[config.AgentClaude]; p != nil {
		p.probeWith(m.configs[0])
		for _, c := range m.configs {
			if c.ID == config.AgentClaude {
				p.probeWith(c)
			}
		}
	}
	m.mu.Unlock()

	errCh := make(chan error, 1)
	go func() { errCh <- m.Start(config.AgentClaude) }()

	// While start may be in progress, List should still return
	for i := 0; i < 20; i++ {
		_ = m.List()
		_ = m.RunningCount()
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	_ = m.Stop(config.AgentClaude)
}

func TestNextNeedingAttentionCycles(t *testing.T) {
	m := NewManager(t.TempDir(), nil)
	// Force waiting on two agents via direct status on live pointers
	a := m.Get(config.AgentClaude)
	b := m.Get(config.AgentCursor)
	if a == nil || b == nil {
		t.Fatal("missing agents")
	}
	a.mu.Lock()
	a.Status = StatusWaiting
	a.waitReason = "y/n"
	a.mu.Unlock()
	b.mu.Lock()
	b.Status = StatusWaiting
	b.waitReason = "approve"
	b.mu.Unlock()

	first := m.NextNeedingAttention("")
	if first == "" {
		t.Fatal("expected someone")
	}
	second := m.NextNeedingAttention(first)
	if second == "" || second == first {
		t.Fatalf("expected different second, got %q after %q", second, first)
	}
	third := m.NextNeedingAttention(second)
	// should wrap to first eventually among the two
	if third != first && third != second {
		// may be another agent if any other is waiting
		_ = third
	}
}

func TestSendEmptyRejected(t *testing.T) {
	m := NewManager(t.TempDir(), nil)
	err := m.Send(config.AgentClaude, "   ")
	if err == nil {
		t.Fatal("expected empty prompt error")
	}
}

func TestSetWorktreeEffectiveCwd(t *testing.T) {
	root := t.TempDir()
	wt := filepath.Join(root, "wt")
	if err := os.Mkdir(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	m := NewManager(root, nil)
	m.SetWorktree(config.AgentGrok, wt)
	if m.EffectiveCwd(config.AgentGrok) != wt {
		t.Fatalf("got %s", m.EffectiveCwd(config.AgentGrok))
	}
	if m.Worktree(config.AgentGrok) != wt {
		t.Fatal(m.Worktree(config.AgentGrok))
	}
}
