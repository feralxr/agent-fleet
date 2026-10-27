package ui

import (
	"os"
	"path/filepath"
	"strings"
)

// expandPath resolves ~, $VAR, and relative paths to an absolute path.
func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	// env vars like $HOME/foo or ${HOME}/foo
	p = os.ExpandEnv(p)
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			p = home
		}
	} else if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
