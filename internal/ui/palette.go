package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type paletteAction struct {
	ID    string
	Label string
	Hint  string
}

var paletteActions = []paletteAction{
	{"start", "Start agent", "s"},
	{"stop", "Stop agent", "x"},
	{"restart", "Restart agent", "r"},
	{"stop-all", "Stop all agents", "X"},
	{"model", "Select model", "m"},
	{"templates", "Prompt templates", "T"},
	{"note", "Edit agent note", "n"},
	{"compare", "Toggle compare mode", "o"},
	{"focus", "Toggle focus mode", "z"},
	{"worktree", "Create worktree", "w"},
	{"diff", "Git diff panel", "g"},
	{"fan", "Fan-out prompt", "f"},
	{"cwd", "Change directory", "c"},
	{"export", "Export output", "e"},
	{"resend", "Resend last prompt", "p"},
	{"split", "Toggle split view", "v"},
	{"jump", "Jump to needing attention", "space"},
	{"reload", "Reload fleet.toml", "ctrl+r"},
}

func filterPalette(query string) []paletteAction {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return paletteActions
	}
	var out []paletteAction
	for _, a := range paletteActions {
		if strings.Contains(strings.ToLower(a.ID), q) ||
			strings.Contains(strings.ToLower(a.Label), q) ||
			strings.Contains(strings.ToLower(a.Hint), q) {
			out = append(out, a)
		}
	}
	return out
}

func (m Model) openPalette() (tea.Model, tea.Cmd) {
	m.paletteMode = true
	m.paletteQuery = ""
	m.paletteIdx = 0
	m.paletteItems = filterPalette("")
	m.statusMsg = "command palette — type to filter, enter run, esc"
	return m, nil
}

func (m Model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.FocusOut):
		m.paletteMode = false
		m.statusMsg = "palette closed"
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.paletteIdx > 0 {
			m.paletteIdx--
		} else if len(m.paletteItems) > 0 {
			m.paletteIdx = len(m.paletteItems) - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.paletteIdx < len(m.paletteItems)-1 {
			m.paletteIdx++
		} else {
			m.paletteIdx = 0
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if len(m.paletteItems) == 0 {
			m.paletteMode = false
			return m, nil
		}
		act := m.paletteItems[m.paletteIdx]
		m.paletteMode = false
		return m.runPaletteAction(act.ID)
	case msg.Type == tea.KeyBackspace:
		if len(m.paletteQuery) > 0 {
			m.paletteQuery = m.paletteQuery[:len(m.paletteQuery)-1]
			m.paletteItems = filterPalette(m.paletteQuery)
			m.paletteIdx = 0
		}
		return m, nil
	case msg.Type == tea.KeyRunes:
		m.paletteQuery += string(msg.Runes)
		m.paletteItems = filterPalette(m.paletteQuery)
		m.paletteIdx = 0
		return m, nil
	}
	return m, nil
}

func (m Model) runPaletteAction(id string) (tea.Model, tea.Cmd) {
	// Dispatch by synthesizing the same paths as keybindings where possible.
	switch id {
	case "start":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	case "stop":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	case "restart":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	case "stop-all":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	case "model":
		return m.openModelPicker()
	case "templates":
		m.tmplMode = true
		m.tmplIdx = 0
		m.statusMsg = "templates"
		return m, nil
	case "note":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	case "compare":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	case "focus":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	case "worktree":
		return m.ensureWorktree()
	case "diff":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	case "fan":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	case "cwd":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	case "export":
		return m.exportOutput()
	case "resend":
		return m.resendLastPrompt()
	case "split":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	case "jump":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeySpace})
	case "reload":
		return m.updateNormal(tea.KeyMsg{Type: tea.KeyCtrlR})
	default:
		m.statusMsg = fmt.Sprintf("unknown action %s", id)
		return m, nil
	}
}

func (m Model) renderPalette() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(" COMMANDS ") + "\n")
	b.WriteString(subtleStyle.Render(" filter: ") + selectedStyle.Render(m.paletteQuery) + "│" + "\n\n")
	if len(m.paletteItems) == 0 {
		b.WriteString(subtleStyle.Render("  no matches") + "\n")
	}
	for i, a := range m.paletteItems {
		prefix := "  "
		line := fmt.Sprintf("%-12s  %-28s  %s", a.ID, a.Label, a.Hint)
		if i == m.paletteIdx {
			prefix = activeBar.Render("▌") + " "
			line = selectedStyle.Render(line)
		} else {
			line = subtleStyle.Render(line)
		}
		b.WriteString(prefix + line + "\n")
	}
	b.WriteString("\n" + subtleStyle.Render(" enter run  ·  esc close"))
	return lipgloss.NewStyle().Width(m.width - 4).Render(b.String())
}
