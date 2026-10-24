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
