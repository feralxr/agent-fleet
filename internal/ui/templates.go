package ui

import "strings"

// Prompt templates for agentic coding workflows.
// Trigger: type /name as the whole prompt, or use the template picker (T).
var promptTemplates = []struct {
	ID    string
	Label string
	Body  string
}{
	{"review", "Code review", "Review the current changes carefully. List bugs, risks, and concrete fixes. Prefer small, actionable comments."},
	{"tests", "Add tests", "Add focused unit tests for the code you just wrote or the selected files. Cover edge cases and failure paths."},
	{"explain", "Explain", "Explain the architecture and data flow of the relevant code. Be concise; use bullet points."},
	{"diff", "Summarize diff", "Summarize the current git diff: intent, risk, and what to test before merge."},
	{"commit", "Commit message", "Write a conventional commit message for the current changes. One subject line + short body."},
	{"refactor", "Safe refactor", "Propose a minimal refactor that improves clarity without changing behavior. Show the plan before editing."},
	{"fix", "Fix issue", "Investigate and fix the issue. Reproduce if possible, then apply the smallest correct patch."},
	{"docs", "Document", "Update or write documentation for the recent changes (README/comments as appropriate)."},
}

func expandTemplate(text string) (string, string, bool) {
	t := strings.TrimSpace(text)
	if !strings.HasPrefix(t, "/") {
		return text, "", false
	}
	name := strings.TrimPrefix(t, "/")
	name = strings.Fields(name)[0]
	name = strings.ToLower(name)
	// allow /review extra context
	rest := strings.TrimSpace(t[len("/"+name):])
	for _, tmpl := range promptTemplates {
		if tmpl.ID == name {
			if rest != "" {
				return tmpl.Body + "\n\nAdditional context:\n" + rest, tmpl.ID, true
			}
			return tmpl.Body, tmpl.ID, true
		}
	}
	return text, "", false
}

func templateByID(id string) (body string, label string, ok bool) {
	for _, tmpl := range promptTemplates {
		if tmpl.ID == id {
			return tmpl.Body, tmpl.Label, true
		}
	}
	return "", "", false
}
