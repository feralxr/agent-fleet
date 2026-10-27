package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPathHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	got := expandPath("~/fleet-test-dir")
	if !strings.HasPrefix(got, home) {
		t.Fatalf("got %q home %q", got, home)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("not abs: %q", got)
	}
}

func TestExpandPathEnv(t *testing.T) {
	t.Setenv("FLEET_TEST_ROOT", "/tmp/fleet-expand")
	got := expandPath("$FLEET_TEST_ROOT/sub")
	if got != "/tmp/fleet-expand/sub" && !strings.Contains(got, "fleet-expand") {
		t.Fatalf("got %q", got)
	}
}

func TestExpandPathEmpty(t *testing.T) {
	if expandPath("  ") != "" {
		t.Fatal("expected empty")
	}
}
