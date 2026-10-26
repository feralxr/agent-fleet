package process

import (
	"testing"
	"time"
)

func TestClassifyAttentionWaiting(t *testing.T) {
	cases := []string{
		"Do you want to apply these changes?",
		"Awaiting approval to run bash",
		"[Y/n]",
		"Press enter to continue",
		"Allow this tool? (y/n)",
		"Permission required to edit file",
		"Choose one of the following options",
	}
	for _, line := range cases {
		st, reason := ClassifyAttentionReason(StatusRunning, []string{line}, time.Now())
		if st != StatusWaiting {
			t.Fatalf("expected waiting for %q, got %s", line, st)
		}
		if reason == "" {
			t.Fatalf("expected reason for %q", line)
		}
	}
}

func TestClassifyAttentionRunning(t *testing.T) {
	lines := []string{"compiling package foo", "ok  github.com/x/y  0.01s"}
	st, reason := ClassifyAttentionReason(StatusRunning, lines, time.Now())
	if st != StatusRunning {
		t.Fatalf("got %s", st)
	}
	if reason != "" {
		t.Fatalf("unexpected reason %q", reason)
	}
}

func TestClassifyAttentionClearsOnFreshOutput(t *testing.T) {
	st, _ := ClassifyAttentionReason(StatusWaiting, []string{"wrote file main.go"}, time.Now())
	if st != StatusRunning {
		t.Fatalf("expected running after fresh output, got %s", st)
	}
}

func TestClassifyAttentionIgnoresFleetAndPromptEcho(t *testing.T) {
	lines := []string{"[fleet] started", "› do you want to continue?"}
	st, _ := ClassifyAttentionReason(StatusRunning, lines, time.Now())
	if st != StatusRunning {
		t.Fatalf("fleet/prompt echo should not trigger waiting, got %s", st)
	}
}
