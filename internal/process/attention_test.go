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
	}
	for _, line := range cases {
		st := ClassifyAttention(StatusRunning, []string{line}, time.Now())
		if st != StatusWaiting {
			t.Fatalf("expected waiting for %q, got %s", line, st)
		}
	}
}

func TestClassifyAttentionRunning(t *testing.T) {
	lines := []string{"compiling package foo", "ok  github.com/x/y  0.01s"}
	st := ClassifyAttention(StatusRunning, lines, time.Now())
	if st != StatusRunning {
		t.Fatalf("got %s", st)
	}
}

func TestClassifyAttentionClearsOnFreshOutput(t *testing.T) {
	// Was waiting; fresh non-prompt output within 800ms → running
	st := ClassifyAttention(StatusWaiting, []string{"wrote file main.go"}, time.Now())
	if st != StatusRunning {
		t.Fatalf("expected running after fresh output, got %s", st)
	}
}
