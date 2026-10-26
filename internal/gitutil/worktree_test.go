package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsRepoFalse(t *testing.T) {
	dir := t.TempDir()
	if IsRepo(dir) {
		t.Fatal("empty dir should not be repo")
	}
}

func TestWorktreeLifecycle(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s (%v)", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")

	if !IsRepo(dir) {
		t.Fatal("expected repo")
	}
	root, err := Root(dir)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := EnsureWorktree(root, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if !IsRepo(wt) {
		t.Fatalf("worktree not a repo: %s", wt)
	}
	// idempotent
	wt2, err := EnsureWorktree(root, "claude")
	if err != nil || wt2 != wt {
		t.Fatalf("reuse failed: %s %v", wt2, err)
	}
	_ = RemoveWorktree(root, "claude")
}

func TestDiffClean(t *testing.T) {
	dir := t.TempDir()
	if DiffStat(dir) != "" {
		t.Fatal("non-repo should be empty diff")
	}
}

func TestStripPaths(t *testing.T) {
	if ShortName("/a/b/claude") != "claude" {
		t.Fatal(ShortName("/a/b/claude"))
	}
}

func TestCurrentBranchAndDiffDirty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s (%v)", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")

	br := CurrentBranch(dir)
	if br == "" {
		t.Fatal("empty branch")
	}

	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat := DiffStat(dir)
	if stat == "" {
		t.Fatal("expected dirty diff stat")
	}
	short := DiffShortOneLiner(dir)
	if short == "" || short == "clean" {
		t.Fatalf("expected dirty shortstat, got %q", short)
	}

	// untracked
	if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat2 := DiffStat(dir)
	if !contains(stat2, "Untracked") && !contains(stat2, "extra.txt") {
		// Untracked section should appear
		if !contains(stat2, "Untracked") {
			t.Fatalf("expected untracked section in %q", stat2)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestEnsureWorktreeBranchReuse(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s (%v)", args, out, err)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")

	wt, err := EnsureWorktree(dir, "cursor")
	if err != nil {
		t.Fatal(err)
	}
	// Remove directory but leave branch so next add hits "already exists"
	_ = RemoveWorktree(dir, "cursor")
	// EnsureWorktree again should succeed even if branch fleet/cursor remains
	wt2, err := EnsureWorktree(dir, "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if !IsRepo(wt2) {
		t.Fatalf("not repo %s", wt2)
	}
	_ = wt
	_ = RemoveWorktree(dir, "cursor")
}

func TestRemoveWorktreeMissingIsNoop(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	// not a repo — should still not panic
	if err := RemoveWorktree(dir, "nope"); err != nil {
		t.Fatal(err)
	}
}
