package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinAgents(t *testing.T) {
	agents := BuiltinAgents()
	if len(agents) != 4 {
		t.Fatalf("expected 4 builtins, got %d", len(agents))
	}
	ids := map[AgentID]bool{}
	for _, a := range agents {
		if a.Command == "" {
			t.Errorf("agent %s has empty command", a.ID)
		}
		ids[a.ID] = true
	}
	for _, id := range []AgentID{AgentClaude, AgentCursor, AgentOpenCode, AgentGrok} {
		if !ids[id] {
			t.Errorf("missing builtin %s", id)
		}
	}
}

func TestResolveAgentsExtra(t *testing.T) {
	file := &FleetFile{
		Extra: []AgentConfig{
			{ID: "aider", Name: "Aider", Command: "aider", UsePTY: BoolPtr(true)},
		},
	}
	agents := ResolveAgents(file, nil)
	if len(agents) != 5 {
		t.Fatalf("expected 5 agents, got %d", len(agents))
	}
	found := false
	for _, a := range agents {
		if a.ID == "aider" {
			found = true
		}
	}
	if !found {
		t.Fatal("extra agent not found")
	}
}

func TestResolveAgentsReplace(t *testing.T) {
	file := &FleetFile{
		Agents: []AgentConfig{
			{ID: "only", Name: "Only", Command: "echo"},
		},
	}
	agents := ResolveAgents(file, nil)
	if len(agents) != 1 || agents[0].ID != "only" {
		t.Fatalf("expected only replacement agent, got %#v", agents)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("FLEET_CLAUDE_CMD", "/custom/claude")
	agents := ResolveAgents(nil, nil)
	for _, a := range agents {
		if a.ID == AgentClaude {
			if a.Command != "/custom/claude" {
				t.Fatalf("env override failed: %s", a.Command)
			}
			return
		}
	}
	t.Fatal("claude not found")
}

func TestSlugify(t *testing.T) {
	if slugify("My Agent!") != "my-agent" {
		t.Fatalf("slugify: %q", slugify("My Agent!"))
	}
}

func TestSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// point config dir via HOME
	t.Setenv("HOME", dir)
	s := Session{LastCwd: "/tmp/proj", LastAgentID: AgentGrok, Worktrees: map[string]string{"claude": "/tmp/wt"}}
	if err := SaveSession(s); err != nil {
		t.Fatal(err)
	}
	got := LoadSession()
	if got.LastCwd != s.LastCwd || got.LastAgentID != s.LastAgentID {
		t.Fatalf("session mismatch: %#v", got)
	}
	if got.Worktrees["claude"] != "/tmp/wt" {
		t.Fatalf("worktrees: %#v", got.Worktrees)
	}
}

func TestLoadFleetFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.toml")
	content := `
[[extra_agents]]
id = "codex"
name = "Codex"
command = "codex"
use_pty = true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FLEET_CONFIG", path)
	file, loaded, err := LoadFleetFile()
	if err != nil {
		t.Fatal(err)
	}
	if loaded != path {
		t.Fatalf("loaded path %s", loaded)
	}
	if len(file.Extra) != 1 || file.Extra[0].ID != "codex" {
		t.Fatalf("extra: %#v", file.Extra)
	}
}

func TestSessionNotesAndPinsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// ConfigDir uses UserHomeDir -> ~/.config/agent-fleet
	s := Session{
		LastCwd: dir,
		Notes:   map[string]string{"claude": "focus on auth"},
		Pins:    map[string]string{"1": "review", "2": "tests"},
		Models:  map[string]string{"claude": "sonnet"},
	}
	if err := SaveSession(s); err != nil {
		t.Fatal(err)
	}
	got := LoadSession()
	if got.Notes["claude"] != "focus on auth" {
		t.Fatalf("notes %v", got.Notes)
	}
	if got.Pins["1"] != "review" || got.Pins["2"] != "tests" {
		t.Fatalf("pins %v", got.Pins)
	}
	if got.Models["claude"] != "sonnet" {
		t.Fatalf("models %v", got.Models)
	}
}

func TestResolveAgentsEnvOverride(t *testing.T) {
	t.Setenv("FLEET_CLAUDE_CMD", "/custom/claude")
	agents := ResolveAgents(nil, nil)
	var found bool
	for _, a := range agents {
		if a.ID == AgentClaude {
			found = true
			if a.Command != "/custom/claude" {
				t.Fatalf("command %s", a.Command)
			}
		}
	}
	if !found {
		t.Fatal("claude missing")
	}
}

func TestResolveAgentsExtraAndDedup(t *testing.T) {
	f := &FleetFile{
		Extra: []AgentConfig{
			{ID: "aider", Name: "Aider", Command: "aider"},
			{ID: "claude", Name: "Claude Override", Command: "claude-custom"},
		},
	}
	agents := ResolveAgents(f, nil)
	var aider, claude *AgentConfig
	for i := range agents {
		if agents[i].ID == "aider" {
			aider = &agents[i]
		}
		if agents[i].ID == AgentClaude {
			claude = &agents[i]
		}
	}
	if aider == nil {
		t.Fatal("extra agent missing")
	}
	if claude == nil || claude.Command != "claude-custom" {
		t.Fatalf("dedup override failed: %+v", claude)
	}
}

func TestConfigDirXDG(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dir, "agent-fleet") {
		t.Fatalf("got %s", got)
	}
}

func TestUsePTYExplicitFalse(t *testing.T) {
	f := false
	file := &FleetFile{
		Agents: []AgentConfig{
			{ID: AgentClaude, Name: "Claude", Command: "claude", UsePTY: &f},
		},
	}
	agents := ResolveAgents(file, nil)
	if len(agents) != 1 {
		t.Fatalf("len %d", len(agents))
	}
	if agents[0].UsePTY == nil || *agents[0].UsePTY {
		t.Fatalf("expected use_pty false, got %v", agents[0].UsePTY)
	}
}

func TestUsePTYUnsetInheritsGlobal(t *testing.T) {
	g := false
	file := &FleetFile{
		UsePTY: &g,
		Agents: []AgentConfig{
			{ID: AgentClaude, Name: "Claude", Command: "claude"}, // UsePTY unset
		},
	}
	agents := ResolveAgents(file, nil)
	if agents[0].UsePTY == nil || *agents[0].UsePTY {
		t.Fatalf("expected inherit false, got %v", agents[0].UsePTY)
	}
}

func TestSaveSessionAtomic(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	s := Session{LastCwd: "/tmp/x", PromptHist: []string{"a", "b"}}
	if err := SaveSession(s); err != nil {
		t.Fatal(err)
	}
	got := LoadSession()
	if got.LastCwd != "/tmp/x" || len(got.PromptHist) != 2 {
		t.Fatalf("%+v", got)
	}
	// no leftover temps
	entries, err := os.ReadDir(filepath.Join(dir, "agent-fleet"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Fatalf("leftover temp %s", e.Name())
		}
	}
}

func TestNamedSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	s := Session{
		Named: map[string]NamedSession{
			"feature-x": {
				Cwd:        dir,
				ActiveID:   AgentClaude,
				Models:     map[string]string{"claude": "sonnet"},
				Notes:      map[string]string{"claude": "auth work"},
				Pins:       map[string]string{"1": "review"},
				PromptHist: []string{"hi"},
			},
		},
	}
	if err := SaveSession(s); err != nil {
		t.Fatal(err)
	}
	got := LoadSession()
	ns, ok := got.Named["feature-x"]
	if !ok {
		t.Fatal("missing named")
	}
	if ns.Models["claude"] != "sonnet" || ns.Notes["claude"] != "auth work" || ns.Pins["1"] != "review" {
		t.Fatalf("%+v", ns)
	}
}
