package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AgentID identifies a supported coding agent.
type AgentID string

const (
	AgentClaude   AgentID = "claude"
	AgentCursor   AgentID = "cursor"
	AgentOpenCode AgentID = "opencode"
	AgentGrok     AgentID = "grok"
)

// AgentConfig holds the launch configuration for one agent.
type AgentConfig struct {
	ID          AgentID
	Name        string
	Command     string
	Args        []string
	Description string
}

// Session holds persisted UI state between runs.
type Session struct {
	LastCwd      string  `json:"last_cwd"`
	LastAgentID  AgentID `json:"last_agent_id"`
	CustomCmds   map[string]string `json:"custom_cmds,omitempty"` // agent id -> command override
}

// DefaultAgents returns the built-in agent definitions.
// Environment variables take highest priority, then session custom cmds, then defaults.
func DefaultAgents() []AgentConfig {
	return DefaultAgentsWithSession(nil)
}

// DefaultAgentsWithSession applies any custom command overrides from the session.
func DefaultAgentsWithSession(s *Session) []AgentConfig {
	agents := []AgentConfig{
		{
			ID:          AgentClaude,
			Name:        "Claude Code",
			Command:     "claude",
			Args:        []string{}, // interactive REPL by default
			Description: "Anthropic Claude Code (claude)",
		},
		{
			ID:          AgentCursor,
			Name:        "Cursor",
			// Prefer `cursor agent` when available; override with FLEET_CURSOR_CMD
			Command:     "cursor",
			Args:        []string{"agent"},
			Description: "Cursor agent CLI",
		},
		{
			ID:          AgentOpenCode,
			Name:        "OpenCode",
			Command:     "opencode",
			Args:        []string{},
			Description: "OpenCode coding agent",
		},
		{
			ID:          AgentGrok,
			Name:        "Grok",
			Command:     "grok",
			Args:        []string{},
			Description: "xAI Grok coding agent",
		},
	}

	for i := range agents {
		id := string(agents[i].ID)
		// 1. Environment variable wins
		envKey := "FLEET_" + toEnvKey(id) + "_CMD"
		if v := os.Getenv(envKey); v != "" {
			agents[i].Command = v
			continue
		}
		// 2. Session custom command
		if s != nil && s.CustomCmds != nil {
			if v, ok := s.CustomCmds[id]; ok && v != "" {
				agents[i].Command = v
			}
		}
	}
	return agents
}

func toEnvKey(id string) string {
	switch id {
	case "claude":
		return "CLAUDE"
	case "cursor":
		return "CURSOR"
	case "opencode":
		return "OPENCODE"
	case "grok":
		return "GROK"
	default:
		return "UNKNOWN"
	}
}

// ConfigDir returns ~/.config/agent-fleet (creating it if needed).
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

// LoadSession reads the last session from disk. Returns a zero Session on error.
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

// SaveSession writes the session to disk.
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
