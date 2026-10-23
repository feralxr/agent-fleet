package gitutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsRepo returns true if path is inside a git working tree.
func IsRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// Root returns the top-level git directory for path.
func Root(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repo: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentBranch returns the current branch name (or detached HEAD short sha).
func CurrentBranch(path string) string {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	b := strings.TrimSpace(string(out))
	if b == "HEAD" {
		cmd = exec.Command("git", "-C", path, "rev-parse", "--short", "HEAD")
		out, err = cmd.Output()
		if err != nil {
			return "detached"
		}
		return "detached-" + strings.TrimSpace(string(out))
	}
	return b
}

// WorktreePath returns the path for an agent's fleet worktree under the repo.
// Layout: <repo>/.fleet/worktrees/<agentID>
func WorktreePath(repoRoot, agentID string) string {
	return filepath.Join(repoRoot, ".fleet", "worktrees", agentID)
}

// EnsureWorktree creates a git worktree for agentID if it does not exist.
// Branch name: fleet/<agentID>
// Returns the worktree absolute path.
func EnsureWorktree(repoRoot, agentID string) (string, error) {
	if !IsRepo(repoRoot) {
		return "", fmt.Errorf("not a git repository: %s", repoRoot)
	}

	wt := WorktreePath(repoRoot, agentID)
	branch := "fleet/" + agentID

	// Already exists and is a directory → reuse
	if st, err := os.Stat(wt); err == nil && st.IsDir() {
		if IsRepo(wt) {
			return wt, nil
		}
		// Broken leftover — remove and recreate
		_ = os.RemoveAll(wt)
	}

	if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
		return "", err
	}

	// Prefer creating from current HEAD
	// git worktree add -b fleet/<id> <path>
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "add", "-b", branch, wt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Branch may already exist from a previous run
		msg := string(out)
		if strings.Contains(msg, "already exists") || strings.Contains(msg, "is not a valid branch") {
			cmd = exec.Command("git", "-C", repoRoot, "worktree", "add", wt, branch)
			out, err = cmd.CombinedOutput()
			if err != nil {
				// Last resort: attach to HEAD without new branch name conflict
				cmd = exec.Command("git", "-C", repoRoot, "worktree", "add", "--detach", wt)
				out, err = cmd.CombinedOutput()
				if err != nil {
					return "", fmt.Errorf("worktree add failed: %s (%w)", strings.TrimSpace(string(out)), err)
				}
			}
		} else {
			return "", fmt.Errorf("worktree add failed: %s (%w)", strings.TrimSpace(msg), err)
		}
	}

	return wt, nil
}

// RemoveWorktree removes a fleet worktree (best-effort).
func RemoveWorktree(repoRoot, agentID string) error {
	wt := WorktreePath(repoRoot, agentID)
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "remove", "--force", wt)
	_ = cmd.Run()
	_ = os.RemoveAll(wt)
	// Prune stale worktree metadata
	_ = exec.Command("git", "-C", repoRoot, "worktree", "prune").Run()
	return nil
}

// ShortName returns a display-friendly worktree label.
func ShortName(wtPath string) string {
	if wtPath == "" {
		return ""
	}
	return filepath.Base(wtPath)
}
