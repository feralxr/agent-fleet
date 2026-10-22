package config

import (
	"os"
	"path/filepath"
)

// AgentID identifies a supported coding agent.
type AgentID string

const (
	AgentClaude  AgentID = "claude"
	AgentCursor  AgentID = "cursor"
	AgentOpenCode AgentID = "opencode"
	AgentGrok    AgentID = "grok"
)

// AgentConfig holds the launch configuration for one agent.
type AgentConfig struct {
	ID          AgentID
	Name        string
	Command     string
	Args        []string
	Description string
}

// DefaultAgents returns the built-in agent definitions.
// Users can override the commands via environment variables later.
func DefaultAgents() []AgentConfig {
	return []AgentConfig{
		{
			ID:          AgentClaude,
			Name:        "Claude Code",
			Command:     envOr("FLEET_CLAUDE_CMD", "claude"),
			Args:        []string{},
			Description: "Anthropic Claude Code agent",
		},
		{
			ID:          AgentCursor,
			Name:        "Cursor",
			Command:     envOr("FLEET_CURSOR_CMD", "cursor"),
			Args:        []string{"agent"},
			Description: "Cursor CLI agent",
		},
		{
			ID:          AgentOpenCode,
			Name:        "OpenCode",
			Command:     envOr("FLEET_OPENCODE_CMD", "opencode"),
			Args:        []string{},
			Description: "OpenCode agent",
		},
		{
			ID:          AgentGrok,
			Name:        "Grok",
			Command:     envOr("FLEET_GROK_CMD", "grok"),
			Args:        []string{},
			Description: "xAI Grok coding agent",
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ConfigDir returns the directory used for fleet config and session data.
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
