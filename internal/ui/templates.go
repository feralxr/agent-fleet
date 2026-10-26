package ui

import "strings"

// Prompt templates for agentic coding workflows.
// Trigger: type /name as the whole prompt, or use the template picker (T).
var promptTemplates = []struct {
	ID    string
	Label string
	Body  string
}{
	{"review", "Code review", `Review the current changes carefully.

For each issue found, report:
1. Severity: blocker | major | minor | nit
2. Location: file and symbol if possible
3. Why it matters
4. Concrete fix (snippet or steps)

Also list what looks solid. Prefer small, actionable comments over style lectures.`},
	{"tests", "Add tests", `Add focused automated tests for the code you just wrote or the files in scope.

Requirements:
- Cover happy path and at least two edge / failure cases
- Prefer table-driven tests where the language supports them
- Name tests after behavior, not implementation
- Run the tests and fix failures before finishing`},
	{"explain", "Explain", `Explain the architecture and data flow of the relevant code.

Structure:
1. Purpose (one paragraph)
2. Main types / modules and responsibilities
3. Request / data flow (bullets)
4. Important invariants or edge cases
5. What a new contributor should read first

Be concise.`},
	{"diff", "Summarize diff", `Summarize the current git diff for a reviewer.

Include:
- Intent of the change (1–2 sentences)
- Risk areas (auth, data loss, concurrency, API breaks)
- Suggested test plan before merge
- Anything that looks unfinished or temporary`},
	{"commit", "Commit message", `Write a conventional commit message for the current changes.

Format:
  <type>(optional scope): <imperative subject ≤72 chars>

  <body: why, not only what; wrap ~72>

Types: feat, fix, refactor, docs, test, chore, perf.
Do not wrap the subject in quotes. Output only the message.`},
	{"refactor", "Safe refactor", `Propose a minimal refactor that improves clarity without changing behavior.

Steps:
1. State the smell / friction in one sentence
2. List the exact files/symbols you will touch
3. Describe the before → after shape
4. Wait for approval before editing if the change is large; otherwise apply and run tests`},
	{"fix", "Fix issue", `Investigate and fix the issue.

Process:
1. Reproduce or locate evidence (error text, failing test, stack)
2. Identify root cause (not just the symptom)
3. Apply the smallest correct patch
4. Add or update a test that would have caught this
5. Summarize cause + fix in 3 bullets`},
	{"docs", "Document", `Update documentation for the recent changes.

Prefer in this order:
1. User-facing README / usage if behavior changed
2. Package or module docs for public APIs
3. Inline comments only where intent is non-obvious

Keep tone factual. Avoid restating the code line-for-line.`},
	{"plan", "Implementation plan", `Before writing code, produce a short implementation plan.

Include:
- Goal and non-goals
- Steps in order (numbered)
- Files likely touched
- Risks / open questions
- How you will verify (tests, manual checks)

Keep it under one screen. Do not start editing until the plan is clear.`},
	{"pr", "PR description", `Draft a pull request description for the current changes.

Sections:
## Summary
## Motivation
## Test plan
## Risks / rollout

Use markdown. Be specific about user-visible behavior.`},
}

func expandTemplate(text string) (string, string, bool) {
	t := strings.TrimSpace(text)
	if !strings.HasPrefix(t, "/") {
		return text, "", false
	}
	name := strings.TrimPrefix(t, "/")
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return text, "", false
	}
	name = strings.ToLower(fields[0])
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
