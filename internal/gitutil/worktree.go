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

// DiffStat returns a short git diff --stat for path (worktree or repo cwd).
// Includes unstaged + staged changes. Empty string if clean or not a repo.
func DiffStat(path string) string {
	if path == "" || !IsRepo(path) {
		return ""
	}
	// Unstaged + untracked summary
	cmd := exec.Command("git", "-C", path, "diff", "--stat", "HEAD")
	out, err := cmd.CombinedOutput()
	stat := strings.TrimSpace(string(out))
	if err != nil && stat == "" {
		// Fallback: unstaged only
		cmd = exec.Command("git", "-C", path, "diff", "--stat")
		out, err = cmd.CombinedOutput()
		stat = strings.TrimSpace(string(out))
	}

	// Also list untracked files briefly
	cmd = exec.Command("git", "-C", path, "ls-files", "--others", "--exclude-standard")
	untracked, _ := cmd.Output()
	ut := strings.TrimSpace(string(untracked))

	var b strings.Builder
	if stat != "" {
		b.WriteString(stat)
	}
	if ut != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		lines := strings.Split(ut, "\n")
		b.WriteString(fmt.Sprintf("Untracked: %d file(s)", len(lines)))
		max := 8
		if len(lines) < max {
			max = len(lines)
		}
		for i := 0; i < max; i++ {
			b.WriteString("\n  + ")
			b.WriteString(lines[i])
		}
		if len(lines) > max {
			b.WriteString(fmt.Sprintf("\n  … +%d more", len(lines)-max))
		}
	}
	return b.String()
}

// DiffShortOneLiner returns a single-line summary suitable for the sidebar.
func DiffShortOneLiner(path string) string {
	if path == "" || !IsRepo(path) {
		return ""
	}
	cmd := exec.Command("git", "-C", path, "diff", "--shortstat", "HEAD")
	out, err := cmd.Output()
	s := strings.TrimSpace(string(out))
	if err != nil || s == "" {
		cmd = exec.Command("git", "-C", path, "status", "--porcelain")
		out, err = cmd.Output()
		if err != nil {
			return ""
		}
		n := 0
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) != "" {
				n++
			}
		}
		if n == 0 {
			return "clean"
		}
		return fmt.Sprintf("%d changed", n)
	}
	// e.g. " 3 files changed, 20 insertions(+), 4 deletions(-)"
	s = strings.ReplaceAll(s, "files changed", "files")
	s = strings.ReplaceAll(s, "file changed", "file")
	s = strings.ReplaceAll(s, "insertions(+)", "+")
	s = strings.ReplaceAll(s, "insertion(+)", "+")
	s = strings.ReplaceAll(s, "deletions(-)", "-")
	s = strings.ReplaceAll(s, "deletion(-)", "-")
	return strings.TrimSpace(s)
}
