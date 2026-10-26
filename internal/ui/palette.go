package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
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
	if len(m.agents) == 0 && id != "reload" && id != "cwd" {
		m.statusMsg = "no agents loaded"
		return m, nil
	}
	switch id {
	case "start":
		id := m.agents[m.selected].ID
		if err := m.manager.Start(id); err != nil {
			m.statusMsg = err.Error()
		} else {
			m.statusMsg = fmt.Sprintf("started %s", m.agents[m.selected].Name)
			m.activeID = id
		}
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil
	case "stop":
		id := m.agents[m.selected].ID
		_ = m.manager.Stop(id)
		m.statusMsg = fmt.Sprintf("stopped %s", m.agents[m.selected].Name)
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil
	case "restart":
		id := m.agents[m.selected].ID
		if err := m.manager.Restart(id); err != nil {
			m.statusMsg = err.Error()
		} else {
			m.statusMsg = fmt.Sprintf("restarted %s", m.agents[m.selected].Name)
		}
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil
	case "stop-all":
		n := m.manager.StopAll()
		m.statusMsg = fmt.Sprintf("stopped %d agents", n)
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil
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
		m.compareMode = !m.compareMode
		if m.compareMode {
			m.split = true
			m.focusMode = false
			m.layout()
			m.outFP = ""
			m.outFP2 = ""
			m.refreshViewport()
			m.statusMsg = "compare on"
		} else {
			m.statusMsg = "compare off"
		}
		return m, nil
	case "focus":
		m.focusMode = !m.focusMode
		m.layout()
		m.statusMsg = map[bool]string{true: "focus on", false: "focus off"}[m.focusMode]
		return m, nil
	case "worktree":
		return m.ensureWorktree()
	case "diff":
		m.showDiff = !m.showDiff
		m.lastContent = ""
		m.refreshViewport()
		m.statusMsg = map[bool]string{true: "diff view", false: "agent output"}[m.showDiff]
		return m, nil
	case "fan":
		m.inputFocus = true
		m.fanMode = true
		m.input.Placeholder = "Fan-out prompt…"
		m.input.Focus()
		m.layout()
		m.statusMsg = "fan-out mode"
		return m, textarea.Blink
	case "cwd":
		m.pathMode = true
		m.inputFocus = true
		m.input.SetValue(m.manager.Cwd())
		m.input.Placeholder = "Working directory…"
		m.input.Focus()
		m.layout()
		return m, textarea.Blink
	case "export":
		return m.exportOutput()
	case "resend":
		return m.resendLastPrompt()
	case "split":
		m.split = !m.split
		m.layout()
		m.outFP = ""
		m.outFP2 = ""
		m.refreshViewport()
		m.statusMsg = map[bool]string{true: "split on", false: "split off"}[m.split]
		return m, nil
	case "jump":
		next := m.manager.NextNeedingAttention(m.activeID)
		if next == "" {
			m.statusMsg = "nothing needs attention"
			return m, nil
		}
		for i, a := range m.agents {
			if a.ID == next {
				m.selected = i
				m.activeID = next
				break
			}
		}
		m.outFP = ""
		m.statusMsg = fmt.Sprintf("jumped to %s", next)
		m.refreshViewport()
		return m, nil
	case "reload":
		added := m.manager.ReloadConfigs(&m.session)
		m.agents = m.manager.List()
		if m.selected >= len(m.agents) {
			m.selected = 0
		}
		if len(m.agents) > 0 {
			m.activeID = m.agents[m.selected].ID
		}
		m.lastContent = ""
		m.statusMsg = fmt.Sprintf("reloaded (%d agents, +%d)", len(m.agents), added)
		m.refreshViewport()
		return m, nil
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
