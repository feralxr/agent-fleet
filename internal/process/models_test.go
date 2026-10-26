package process

import "testing"

func TestParseModelLines(t *testing.T) {
	in := `
Available models
anthropic/claude-sonnet-4-5
openai/gpt-5 (default)
• grok-4
Usage: something
`
	got := parseModelLines(in)
	if len(got) < 3 {
		t.Fatalf("got %v", got)
	}
	if got[0] != "anthropic/claude-sonnet-4-5" {
		t.Fatal(got[0])
	}
}
