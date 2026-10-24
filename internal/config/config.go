package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// AgentID identifies a coding agent.
type AgentID string

const (
	AgentClaude   AgentID = "claude"
	AgentCursor   AgentID = "cursor"
	AgentOpenCode AgentID = "opencode"
	AgentGrok     AgentID = "grok"
)

// AgentConfig holds launch configuration for one agent.
type AgentConfig struct {
	ID          AgentID  `toml:"id" json:"id"`
	Name        string   `toml:"name" json:"name"`
	Command     string   `toml:"command" json:"command"`
	Args        []string `toml:"args" json:"args"`
	Description string   `toml:"description" json:"description"`
	UsePTY      bool     `toml:"use_pty" json:"use_pty"`
}

// FleetFile is the on-disk fleet.toml schema.
type FleetFile struct {
	DefaultCwd string        `toml:"default_cwd"`
	UsePTY     *bool         `toml:"use_pty"`
	Agents     []AgentConfig `toml:"agents"`
	Extra      []AgentConfig `toml:"extra_agents"`
}

// Session holds persisted UI state between runs.
type Session struct {
	LastCwd     string            `json:"last_cwd"`
	LastAgentID AgentID           `json:"last_agent_id"`
	CustomCmds  map[string]string `json:"custom_cmds,omitempty"`
	Worktrees   map[string]string `json:"worktrees,omitempty"`
	PromptHist  []string          `json:"prompt_history,omitempty"`
	Named       map[string]NamedSession `json:"named_sessions,omitempty"`
}

// NamedSession is a saved layout the user can restore later.
type NamedSession struct {
	Cwd         string            `json:"cwd"`
	ActiveID    AgentID       `json:"active_id"`
	Worktrees   map[string]string `json:"worktrees,omitempty"`
	PromptHist  []string          `json:"prompt_history,omitempty"`
}

// LoadFleetFile reads fleet.toml from search paths.
func LoadFleetFile() (*FleetFile, string, error) {
	candidates := []string{}
	if v := os.Getenv("FLEET_CONFIG"); v != "" {
		candidates = append(candidates, v)
	}
	candidates = append(candidates, "fleet.toml")
	if dir, err := ConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(dir, "fleet.toml"))
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var f FleetFile
		if _, err := toml.Decode(string(data), &f); err != nil {
			return nil, path, fmt.Errorf("parse %s: %w", path, err)
		}
		return &f, path, nil
	}
	return &FleetFile{}, "", nil
}

// BuiltinAgents returns the default four agents.
func BuiltinAgents() []AgentConfig {
	return []AgentConfig{
		{ID: AgentClaude, Name: "Claude Code", Command: "claude", Description: "Anthropic Claude Code", UsePTY: true},
		{ID: AgentCursor, Name: "Cursor", Command: "cursor", Args: []string{"agent"}, Description: "Cursor agent CLI", UsePTY: true},
		{ID: AgentOpenCode, Name: "OpenCode", Command: "opencode", Description: "OpenCode coding agent", UsePTY: true},
		{ID: AgentGrok, Name: "Grok", Command: "grok", Description: "xAI Grok coding agent", UsePTY: true},
	}
}

// ResolveAgents merges built-ins, fleet.toml, env overrides, and session custom cmds.
func ResolveAgents(file *FleetFile, session *Session) []AgentConfig {
	if file == nil {
		file = &FleetFile{}
	}
	globalPTY := true
	if file.UsePTY != nil {
		globalPTY = *file.UsePTY
	}

	var agents []AgentConfig
	if len(file.Agents) > 0 {
		agents = append(agents, file.Agents...)
	} else {
		agents = append(agents, BuiltinAgents()...)
	}
	agents = append(agents, file.Extra...)

	seen := map[AgentID]bool{}
	out := make([]AgentConfig, 0, len(agents))
	for _, a := range agents {
		if a.ID == "" {
			a.ID = AgentID(slugify(a.Name))
		}
		if a.Name == "" {
			a.Name = string(a.ID)
		}
		if a.Command == "" {
			continue
		}
		if isBuiltinID(a.ID) && !a.UsePTY {
			a.UsePTY = globalPTY
		}
		envKey := "FLEET_" + strings.ToUpper(strings.ReplaceAll(string(a.ID), "-", "_")) + "_CMD"
		if v := os.Getenv(envKey); v != "" {
			a.Command = v
			a.Args = nil
		}
		if session != nil && session.CustomCmds != nil {
			if v, ok := session.CustomCmds[string(a.ID)]; ok && v != "" {
				a.Command = v
			}
		}
		if seen[a.ID] {
			for i := range out {
				if out[i].ID == a.ID {
					out[i] = a
					break
				}
			}
			continue
		}
		seen[a.ID] = true
		out = append(out, a)
	}
	return out
}

func DefaultAgents() []AgentConfig {
	return ResolveAgents(nil, nil)
}

func DefaultAgentsWithSession(s *Session) []AgentConfig {
	file, _, _ := LoadFleetFile()
	return ResolveAgents(file, s)
}

func isBuiltinID(id AgentID) bool {
	switch id {
	case AgentClaude, AgentCursor, AgentOpenCode, AgentGrok:
		return true
	default:
		return false
	}
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "agent"
	}
	return b.String()
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "agent-fleet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func sessionPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

func LoadSession() Session {
	path, err := sessionPath()
	if err != nil {
		return Session{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}
	}
	return s
}

func SaveSession(s Session) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func WriteExampleFleetToml(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("already exists: %s", path)
	}
	example := `# Agent Fleet configuration
# ./fleet.toml or ~/.config/agent-fleet/fleet.toml
# Or FLEET_CONFIG=/path/to/fleet.toml

# use_pty = true

# Replace built-ins by defining [[agents]], or keep them and add:

[[extra_agents]]
id = "aider"
name = "Aider"
command = "aider"
args = ["--no-git"]
description = "Aider pair programming"
use_pty = true

[[extra_agents]]
id = "codex"
name = "Codex"
command = "codex"
use_pty = true
`
	return os.WriteFile(path, []byte(example), 0o644)
}
