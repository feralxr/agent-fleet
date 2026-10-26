package process

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/feralxr/agent-fleet/internal/config"
)

// DiscoverModels asks the agent CLI for available models (best-effort, short timeout).
// Returns nil if the CLI has no list command or fails — callers should fall back to defaults.
func DiscoverModels(cfg config.AgentConfig) []string {
	if cfg.Command == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch cfg.ID {
	case config.AgentOpenCode:
		cmd = exec.CommandContext(ctx, cfg.Command, "models")
	case config.AgentGrok:
		cmd = exec.CommandContext(ctx, cfg.Command, "models")
	case config.AgentCursor:
		// cursor agent models  OR  cursor-agent models
		if cfg.Command == "cursor" || (len(cfg.Args) > 0 && cfg.Args[0] == "agent") {
			args := append([]string{}, cfg.Args...)
			args = append(args, "models")
			cmd = exec.CommandContext(ctx, cfg.Command, args...)
		} else {
			cmd = exec.CommandContext(ctx, cfg.Command, "models")
		}
	case config.AgentClaude:
		// Claude Code has no stable "models" subcommand; aliases only.
		return nil
	default:
		// Try common patterns for custom agents
		cmd = exec.CommandContext(ctx, cfg.Command, "models")
	}
	if cmd == nil {
		return nil
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		// Retry cursor with --list-models
		if cfg.ID == config.AgentCursor {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel2()
			args := append([]string{}, cfg.Args...)
			args = append(args, "--list-models")
			cmd2 := exec.CommandContext(ctx2, cfg.Command, args...)
			var buf2 bytes.Buffer
			cmd2.Stdout = &buf2
			cmd2.Stderr = &buf2
			if err2 := cmd2.Run(); err2 != nil {
				return nil
			}
			return parseModelLines(buf2.String())
		}
		return nil
	}
	return parseModelLines(buf.String())
}

func parseModelLines(out string) []string {
	seen := map[string]bool{}
	var models []string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// Skip obvious headers / noise
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "available") ||
			strings.HasPrefix(low, "model") && strings.Contains(low, "list") ||
			strings.HasPrefix(low, "usage:") ||
			strings.HasPrefix(low, "error") ||
			strings.HasPrefix(line, "─") ||
			strings.HasPrefix(line, "=") {
			continue
		}
		// Take first token; strip bullets and markers
		line = strings.TrimLeft(line, "•*-›> \t")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		id := fields[0]
		// Strip trailing punctuation / (default) markers
		id = strings.Trim(id, ",;")
		if i := strings.Index(id, "("); i > 0 {
			id = strings.TrimSpace(id[:i])
		}
		if len(id) < 2 || len(id) > 80 {
			continue
		}
		// Must look like a model id: letters, digits, / - . _
		ok := true
		for _, r := range id {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
				r == '/' || r == '-' || r == '.' || r == '_' || r == '[' || r == ']') {
				ok = false
				break
			}
		}
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		models = append(models, id)
		if len(models) >= 80 {
			break
		}
	}
	return models
}

// MergeModels prefers live discovery, then config override, then built-in defaults.
func MergeModels(id config.AgentID, cfg *config.AgentConfig, live []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(list []string) {
		for _, m := range list {
			m = strings.TrimSpace(m)
			if m == "" || seen[m] {
				continue
			}
			seen[m] = true
			out = append(out, m)
		}
	}
	add(live)
	if cfg != nil && len(cfg.Models) > 0 {
		add(cfg.Models)
	} else {
		add(config.DefaultModels(id))
	}
	return out
}
