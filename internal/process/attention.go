package process

import (
	"regexp"
	"strings"
	"time"
)

// Heuristics for when an agent is blocked on the user.
var waitingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(awaiting|waiting for)\b.*\b(input|approval|confirm|permission|response)\b`),
	regexp.MustCompile(`(?i)\b(do you want to|shall i|should i|continue\?|proceed\?|apply these changes\?)\b`),
	regexp.MustCompile(`(?i)\b(y/n|yes/no|\[y/n\]|\(y/n\)|\[Y/n\]|\[y/N\])\b`),
	regexp.MustCompile(`(?i)\b(press enter|hit enter|confirm to|type ['\"]?(yes|y|ok)['\"]? to)\b`),
	regexp.MustCompile(`(?i)\b(permission required|needs? approval|awaiting approval)\b`),
	regexp.MustCompile(`(?i)^\s*>\s*$`),                         // bare prompt
	regexp.MustCompile(`(?i)\b(allow|deny|always allow)\b.*\?`), // tool permission prompts
}

// ClassifyAttention scans recent output lines and returns waiting=true when
// the agent appears blocked on human input. Only applies while running.
func ClassifyAttention(status Status, recent []string, lastActive time.Time) Status {
	if status != StatusRunning && status != StatusWaiting {
		return status
	}
	// Look at last ~12 non-empty lines
	var lines []string
	for i := len(recent) - 1; i >= 0 && len(lines) < 12; i-- {
		s := strings.TrimSpace(recent[i])
		if s == "" || strings.HasPrefix(s, "[fleet]") {
			continue
		}
		lines = append(lines, s)
	}
	// Most recent first in lines
	for _, line := range lines {
		for _, re := range waitingPatterns {
			if re.MatchString(line) {
				return StatusWaiting
			}
		}
	}
	// If we were waiting but new output arrived recently without match, clear
	if status == StatusWaiting {
		if !lastActive.IsZero() && time.Since(lastActive) < 800*time.Millisecond {
			return StatusRunning
		}
		// Keep waiting if idle with no new contradicting evidence
		return StatusWaiting
	}
	return StatusRunning
}
