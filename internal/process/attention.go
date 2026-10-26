package process

import (
	"regexp"
	"strings"
	"time"
)

// waitingPatterns: ordered from specific → general.
var waitingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(awaiting|waiting for)\b.*\b(input|approval|confirm|permission|response|answer)\b`),
	regexp.MustCompile(`(?i)\b(permission required|needs? (your )?approval|awaiting approval|tool permission)\b`),
	regexp.MustCompile(`(?i)\b(allow|deny|always allow|allow once)\b.*\?`),
	regexp.MustCompile(`(?i)\b(do you want to|shall i|should i|may i|can i go ahead)\b`),
	regexp.MustCompile(`(?i)\b(continue\?|proceed\?|apply these changes\?|accept\?|confirm\?)\b`),
	regexp.MustCompile(`(?i)\b(y/n|yes/no|\[y/n\]|\(y/n\)|\[Y/n\]|\[y/N\]|\(yes/no\))\b`),
	regexp.MustCompile(`(?i)\b(press enter|hit enter|confirm to|type ['"]?(yes|y|ok|continue)['"]? to)\b`),
	regexp.MustCompile(`(?i)\b(choose one|select an option|enter a number|pick a number)\b`),
	regexp.MustCompile(`(?i)^\s*(human|user|you):\s*$`),
	regexp.MustCompile(`(?i)^\s*>\s*$`),
}

// ClassifyAttention scans recent output and returns waiting when the agent
// looks blocked on human input. Also returns a short reason for the UI.
func ClassifyAttention(status Status, recent []string, lastActive time.Time) Status {
	st, _ := ClassifyAttentionReason(status, recent, lastActive)
	return st
}

// ClassifyAttentionReason is the detailed form used by the process layer.
func ClassifyAttentionReason(status Status, recent []string, lastActive time.Time) (Status, string) {
	if status != StatusRunning && status != StatusWaiting {
		return status, ""
	}
	var lines []string
	for i := len(recent) - 1; i >= 0 && len(lines) < 16; i-- {
		s := strings.TrimSpace(recent[i])
		if s == "" || strings.HasPrefix(s, "[fleet]") || strings.HasPrefix(s, "› ") {
			continue
		}
		// Collapse very long lines for matching
		if len(s) > 240 {
			s = s[:240]
		}
		lines = append(lines, s)
	}
	for _, line := range lines {
		for _, re := range waitingPatterns {
			if re.MatchString(line) {
				reason := line
				if len(reason) > 48 {
					reason = reason[:45] + "…"
				}
				return StatusWaiting, reason
			}
		}
	}
	if status == StatusWaiting {
		if !lastActive.IsZero() && time.Since(lastActive) < 800*time.Millisecond {
			return StatusRunning, ""
		}
		return StatusWaiting, "still waiting"
	}
	return StatusRunning, ""
}
