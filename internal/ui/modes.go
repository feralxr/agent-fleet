package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/feralxr/agent-fleet/internal/config"
	"github.com/feralxr/agent-fleet/internal/gitutil"
	"github.com/feralxr/agent-fleet/internal/process"
)

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.modelMode {
		return m.updateModelPicker(msg)
	}
	if m.tmplMode {
		return m.updateTemplatePicker(msg)
	}
	if m.paletteMode {
		return m.updatePalette(msg)
	}
	if m.pinArm && key.Matches(msg, m.keys.FocusOut) {
		m.pinArm = false
		m.statusMsg = "pin cancelled"
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Quit):
		for _, a := range m.agents {
			_ = m.manager.Stop(a.ID)
		}
		m.persistSession()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if len(m.agents) == 0 {
			return m, nil
		}
		if m.selected > 0 {
			m.selected--
		} else {
			m.selected = len(m.agents) - 1 // wrap
		}
		m.activeID = m.agents[m.selected].ID
		m.outFP = ""
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if len(m.agents) == 0 {
			return m, nil
		}
		if m.selected < len(m.agents)-1 {
			m.selected++
		} else {
			m.selected = 0 // wrap
		}
		m.activeID = m.agents[m.selected].ID
		m.outFP = ""
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Start):
		id := m.agents[m.selected].ID
		if err := m.manager.Start(id); err != nil {
			m.statusMsg = fmt.Sprintf("start failed: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("started %s", m.agents[m.selected].Name)
			m.activeID = id
		}
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Stop):
		id := m.agents[m.selected].ID
		_ = m.manager.Stop(id)
		m.statusMsg = fmt.Sprintf("stopped %s", m.agents[m.selected].Name)
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Restart):
		id := m.agents[m.selected].ID
		if err := m.manager.Restart(id); err != nil {
			m.statusMsg = fmt.Sprintf("restart failed: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("restarted %s", m.agents[m.selected].Name)
			m.activeID = id
		}
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Clear):
		id := m.agents[m.selected].ID
		m.manager.ClearOutput(id)
		m.statusMsg = "output cleared"
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Cwd):
		m.pathMode = true
		m.inputFocus = true
		m.input.SetValue(m.manager.Cwd())
		m.input.Placeholder = "Enter working directory path…"
		m.input.Focus()
		m.statusMsg = "change directory — Enter to confirm, Esc to cancel"
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Split):
		m.split = !m.split
		if m.split {
			m.statusMsg = "split view on"
		} else {
			m.statusMsg = "split view off"
		}
		m.layout()
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Worktree):
		return m.ensureWorktree()

	case key.Matches(msg, m.keys.Fan):
		m.inputFocus = true
		m.input.Placeholder = "Fan-out: type prompt, Enter sends to ALL running agents…"
		m.input.Focus()
		m.statusMsg = "fan-out mode — prompt will go to every running agent"
		m.fanMode = true
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Jump):
		next := m.manager.NextNeedingAttention(m.activeID)
		if next == "" {
			m.statusMsg = "no agent needs attention"
			return m, nil
		}
		for i, a := range m.agents {
			if a.ID == next {
				m.selected = i
				m.activeID = next
				break
			}
		}
		m.statusMsg = fmt.Sprintf("jumped to %s", m.agents[m.selected].Name)
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Diff):
		m.showDiff = !m.showDiff
		m.lastContent = ""
		m.lastContent2 = ""
		m.outFP = ""
		if m.showDiff {
			m.refreshDiffCache(m.manager.EffectiveCwd(m.activeID))
			m.statusMsg = "diff view on — press g to return"
		} else {
			m.statusMsg = "diff view off"
		}
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.Reload):
		added := m.manager.ReloadConfigs(&m.session)
		m.agents = m.manager.List()
		if m.selected >= len(m.agents) {
			m.selected = 0
		}
		if len(m.agents) > 0 {
			m.activeID = m.agents[m.selected].ID
		}
		m.lastContent = ""
		m.lastContent2 = ""
		m.statusMsg = fmt.Sprintf("reloaded fleet.toml (%d agents, +%d new)", len(m.agents), added)
		m.refreshViewport()
		return m, nil

	case key.Matches(msg, m.keys.SaveName):
		m.nameMode = true
		m.inputFocus = true
		m.input.SetValue("")
		m.input.Placeholder = "Name this session and press Enter…"
		m.input.Focus()
		m.statusMsg = "save named session"
		return m, textarea.Blink

	case key.Matches(msg, m.keys.LoadName):
		return m.loadNamedSessionCycle()

	case key.Matches(msg, m.keys.PgUp):
		m.viewport.HalfViewUp()
		m.viewport.HalfViewUp()
		return m, nil
	case key.Matches(msg, m.keys.PgDown):
		m.viewport.HalfViewDown()
		m.viewport.HalfViewDown()
		return m, nil
	case key.Matches(msg, m.keys.HalfUp):
		m.viewport.HalfViewUp()
		return m, nil
	case key.Matches(msg, m.keys.HalfDown):
		m.viewport.HalfViewDown()
		return m, nil

	case msg.String() >= "1" && msg.String() <= "9":
		n := int(msg.String()[0] - '0')
		if m.pinArm && n >= 1 && n <= 4 {
			m.pinArm = false
			slot := msg.String()
			id := ""
			if m.session.Pins != nil {
				id = m.session.Pins[slot]
			}
			if id == "" {
				m.statusMsg = fmt.Sprintf("pin %s empty — T then %s to pin a template", slot, slot)
				return m, nil
			}
			body, label, ok := templateByID(id)
			if !ok {
				m.statusMsg = fmt.Sprintf("pin %s invalid: %s", slot, id)
				return m, nil
			}
			m.inputFocus = true
			m.input.SetValue(body)
			m.input.Focus()
			m.layout()
			m.statusMsg = fmt.Sprintf("pin %s · /%s (%s)", slot, id, label)
			return m, textarea.Blink
		}
		m.pinArm = false
		if n >= 1 && n <= len(m.agents) {
			m.selected = n - 1
			m.activeID = m.agents[m.selected].ID
			m.outFP = ""
			m.statusMsg = fmt.Sprintf("selected %s", m.agents[m.selected].Name)
			m.refreshViewport()
		}
		return m, nil

	case key.Matches(msg, m.keys.ModelSel):
		return m.openModelPicker()

	case key.Matches(msg, m.keys.Resend):
		return m.resendLastPrompt()

	case key.Matches(msg, m.keys.Export):
		return m.exportOutput()

	case key.Matches(msg, m.keys.ToggleTarget):
		if len(m.agents) == 0 {
			return m, nil
		}
		id := m.agents[m.selected].ID
		if m.fanTargets == nil {
			m.fanTargets = make(map[config.AgentID]bool)
		}
		if m.fanTargets[id] {
			delete(m.fanTargets, id)
			m.statusMsg = fmt.Sprintf("fan target off · %s", m.agents[m.selected].Name)
		} else {
			m.fanTargets[id] = true
			m.statusMsg = fmt.Sprintf("fan target on · %s (%d selected)", m.agents[m.selected].Name, len(m.fanTargets))
		}
		return m, nil

	case key.Matches(msg, m.keys.Templates):
		m.tmplMode = true
		m.tmplIdx = 0
		m.statusMsg = "templates — j/k · enter insert · esc"
		return m, nil

	case key.Matches(msg, m.keys.FocusMode):
		m.focusMode = !m.focusMode
		m.layout()
		if m.focusMode {
			m.statusMsg = "focus mode on — z to exit"
		} else {
			m.statusMsg = "focus mode off"
		}
		return m, nil

	case key.Matches(msg, m.keys.Note):
		if len(m.agents) == 0 {
			return m, nil
		}
		id := m.agents[m.selected].ID
		cur := ""
		if m.session.Notes != nil {
			cur = m.session.Notes[string(id)]
		}
		m.noteMode = true
		m.inputFocus = true
		m.input.SetValue(cur)
		m.input.Placeholder = "Note for this agent — Enter save, Esc cancel"
		m.input.Focus()
		m.layout()
		m.statusMsg = fmt.Sprintf("note · %s", m.agents[m.selected].Name)
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Compare):
		m.compareMode = !m.compareMode
		if m.compareMode {
			m.split = true
			m.focusMode = false
			m.layout()
			m.outFP = ""
			m.outFP2 = ""
			m.refreshViewport()
			m.statusMsg = "compare on — selected + neighbor"
		} else {
			m.statusMsg = "compare off"
		}
		return m, nil

	case key.Matches(msg, m.keys.Palette):
		return m.openPalette()

	case key.Matches(msg, m.keys.PinFire):
		m.pinArm = true
		parts := []string{}
		if m.session.Pins != nil {
			for _, k := range []string{"1", "2", "3", "4"} {
				if v := m.session.Pins[k]; v != "" {
					parts = append(parts, k+"="+v)
				}
			}
		}
		summary := "empty"
		if len(parts) > 0 {
			summary = strings.Join(parts, " ")
		}
		m.statusMsg = fmt.Sprintf("pin 1-4 (%s)", summary)
		return m, nil

	case key.Matches(msg, m.keys.FocusIn):
		if m.pathMode || m.modelMode {
			return m, nil
		}
		m.inputFocus = true
		m.input.Placeholder = "Type a prompt and press Enter to send to the active agent…"
		m.input.Focus()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Copy):
		p := m.manager.Get(m.activeID)
		if p == nil {
			m.statusMsg = "nothing to copy"
			return m, nil
		}
		lines := p.SnapshotOutput()
		if len(lines) == 0 {
			m.statusMsg = "nothing to copy"
			return m, nil
		}
		// Copy last 200 lines max to keep clipboard reasonable.
		start := 0
		if len(lines) > 200 {
			start = len(lines) - 200
		}
		text := strings.Join(lines[start:], "\n")
		if err := clipboard.WriteAll(text); err != nil {
			m.statusMsg = fmt.Sprintf("copy failed: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("copied %d lines", len(lines)-start)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.FocusOut):
		m.inputFocus = false
		m.pathMode = false
		m.fanMode = false
		m.nameMode = false
		m.histIdx = -1
		m.input.Blur()
		m.input.Reset()
		m.input.Placeholder = "i  focus  ·  enter  send  ·  esc  cancel"
		m.layout()
		m.statusMsg = "cancelled"
		return m, nil

	case key.Matches(msg, m.keys.HistUp):
		if m.pathMode || m.nameMode {
			break
		}
		hist := m.session.PromptHist
		if len(hist) == 0 {
			return m, nil
		}
		if m.histIdx < 0 {
			m.histIdx = len(hist) - 1
		} else if m.histIdx > 0 {
			m.histIdx--
		}
		m.input.SetValue(hist[m.histIdx])
		m.input.CursorEnd()
		return m, nil

	case key.Matches(msg, m.keys.HistDown):
		if m.pathMode || m.nameMode {
			break
		}
		hist := m.session.PromptHist
		if m.histIdx < 0 {
			return m, nil
		}
		if m.histIdx >= len(hist)-1 {
			m.histIdx = -1
			m.input.SetValue("")
			return m, nil
		}
		m.histIdx++
		m.input.SetValue(hist[m.histIdx])
		m.input.CursorEnd()
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}

		if m.noteMode {
			id := m.agents[m.selected].ID
			if m.session.Notes == nil {
				m.session.Notes = map[string]string{}
			}
			if text == "" {
				delete(m.session.Notes, string(id))
				m.statusMsg = "note cleared"
			} else {
				m.session.Notes[string(id)] = text
				m.statusMsg = fmt.Sprintf("note saved · %s", m.agents[m.selected].Name)
			}
			_ = config.SaveSession(m.session)
			m.noteMode = false
			m.inputFocus = false
			m.input.Blur()
			m.input.Reset()
			m.input.Placeholder = "i  focus  ·  enter  send  ·  esc  cancel"
			m.layout()
			return m, nil
		}

		if m.modelCustom {
			id := m.agents[m.selected].ID
			if err := m.manager.ApplyModel(id, text); err != nil {
				m.statusMsg = fmt.Sprintf("set %s, live switch: %v", text, err)
			} else {
				st := m.manager.Get(id)
				if st != nil && st.IsLive() {
					m.statusMsg = fmt.Sprintf("%s → %s (sent /model)", m.agents[m.selected].Name, text)
				} else {
					m.statusMsg = fmt.Sprintf("%s → %s (next start)", m.agents[m.selected].Name, text)
				}
			}
			m.modelCustom = false
			m.inputFocus = false
			m.input.Blur()
			m.input.Reset()
			m.input.Placeholder = "i  focus  ·  enter  send  ·  esc  cancel"
			m.layout()
			m.agents = m.manager.List()
			m.refreshViewport()
			return m, nil
		}

		if m.nameMode {
			m.saveNamedSession(text)
			m.nameMode = false
			m.inputFocus = false
			m.input.Blur()
			m.input.Reset()
			m.input.Placeholder = "Type a prompt and press Enter to send to the active agent…"
			return m, nil
		}

		if m.pathMode {
			// Change working directory
			st, err := os.Stat(text)
			if err != nil || !st.IsDir() {
				m.statusMsg = fmt.Sprintf("invalid directory: %s", text)
				return m, nil
			}
			abs, err := filepath.Abs(text)
			if err != nil {
				abs = text
			}
			m.manager.SetCwd(abs)
			m.session.LastCwd = abs
			_ = config.SaveSession(m.session)
			m.pathMode = false
			m.inputFocus = false
			m.input.Blur()
			m.input.Reset()
			m.input.Placeholder = "Type a prompt and press Enter to send to the active agent…"
			m.statusMsg = fmt.Sprintf("cwd → %s", abs)
			return m, nil
		}

		if m.fanMode {
			if expanded, name, ok := expandTemplate(text); ok {
				text = expanded
				_ = name
			}
			m.pushPrompt(text)
			n := 0
			var lastErr error
			var names []string
			useTargets := len(m.fanTargets) > 0
			for _, a := range m.agents {
				if a.Status != process.StatusRunning && a.Status != process.StatusWaiting {
					continue
				}
				if useTargets && !m.fanTargets[a.ID] {
					continue
				}
				if err := m.manager.Send(a.ID, text); err != nil {
					lastErr = err
				} else {
					n++
					names = append(names, a.Name)
				}
			}
			m.fanMode = false
			m.histIdx = -1
			m.inputFocus = false
			m.input.Blur()
			m.input.Reset()
			m.input.Placeholder = "Type a prompt and press Enter to send to the active agent…"
			if n == 0 {
				m.statusMsg = "fan-out: no matching running agents"
			} else {
				list := strings.Join(names, ", ")
				if len(list) > 40 {
					list = list[:37] + "…"
				}
				if lastErr != nil {
					m.statusMsg = fmt.Sprintf("fan-out %d [%s] (err: %v)", n, list, lastErr)
				} else {
					m.statusMsg = fmt.Sprintf("fan-out %d [%s]", n, list)
				}
			}
			m.refreshViewport()
			return m, nil
		}

		if expanded, name, ok := expandTemplate(text); ok {
			text = expanded
			m.statusMsg = fmt.Sprintf("template /%s → sent", name)
		}
		m.pushPrompt(text)
		err := m.manager.Send(m.activeID, text)
		if err != nil {
			m.statusMsg = fmt.Sprintf("send failed: %v", err)
		} else {
			if !strings.HasPrefix(m.statusMsg, "template") {
				m.statusMsg = "prompt sent"
			}
			m.input.Reset()
			m.histIdx = -1
		}
		m.refreshViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) persistSession() {
	m.session.LastCwd = m.manager.Cwd()
	m.session.LastAgentID = m.activeID
	m.session.Worktrees = map[string]string{}
	m.session.Models = map[string]string{}
	// Notes and Pins are mutated in place on m.session — keep them.
	for _, a := range m.agents {
		if wt := m.manager.Worktree(a.ID); wt != "" {
			m.session.Worktrees[string(a.ID)] = wt
		}
		if mod := m.manager.GetModel(a.ID); mod != "" {
			m.session.Models[string(a.ID)] = mod
		}
	}
	// Drop empty maps to keep session.json small
	if len(m.session.Worktrees) == 0 {
		m.session.Worktrees = nil
	}
	if len(m.session.Models) == 0 {
		m.session.Models = nil
	}
	if len(m.session.Notes) == 0 {
		m.session.Notes = nil
	}
	if len(m.session.Pins) == 0 {
		m.session.Pins = nil
	}
	_ = config.SaveSession(m.session)
}

func (m *Model) pushPrompt(text string) {
	const maxHist = 50
	// Dedupe consecutive
	if len(m.session.PromptHist) > 0 && m.session.PromptHist[len(m.session.PromptHist)-1] == text {
		return
	}
	m.session.PromptHist = append(m.session.PromptHist, text)
	if len(m.session.PromptHist) > maxHist {
		m.session.PromptHist = m.session.PromptHist[len(m.session.PromptHist)-maxHist:]
	}
	_ = config.SaveSession(m.session)
}

func (m *Model) saveNamedSession(name string) {
	if m.session.Named == nil {
		m.session.Named = map[string]config.NamedSession{}
	}
	wts := map[string]string{}
	for _, a := range m.agents {
		if wt := m.manager.Worktree(a.ID); wt != "" {
			wts[string(a.ID)] = wt
		}
	}
	m.session.Named[name] = config.NamedSession{
		Cwd:        m.manager.Cwd(),
		ActiveID:   m.activeID,
		Worktrees:  wts,
		PromptHist: append([]string{}, m.session.PromptHist...),
	}
	_ = config.SaveSession(m.session)
	m.statusMsg = fmt.Sprintf("saved session %q", name)
}

func (m Model) loadNamedSessionCycle() (tea.Model, tea.Cmd) {
	if m.session.Named == nil || len(m.session.Named) == 0 {
		m.statusMsg = "no named sessions — press S to save one"
		return m, nil
	}
	// Deterministic order
	names := make([]string, 0, len(m.session.Named))
	for n := range m.session.Named {
		names = append(names, n)
	}
	// sort
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	// Pick next after current status hint or first
	pick := names[0]
	if strings.HasPrefix(m.statusMsg, "loaded session ") {
		cur := strings.TrimPrefix(m.statusMsg, "loaded session ")
		cur = strings.Trim(cur, `"`)
		for i, n := range names {
			if n == cur && i+1 < len(names) {
				pick = names[i+1]
				break
			}
		}
	}
	ns := m.session.Named[pick]
	if ns.Cwd != "" {
		if st, err := os.Stat(ns.Cwd); err == nil && st.IsDir() {
			m.manager.SetCwd(ns.Cwd)
			m.session.LastCwd = ns.Cwd
		}
	}
	if ns.Worktrees != nil {
		m.session.Worktrees = ns.Worktrees
		for id, wt := range ns.Worktrees {
			m.manager.SetWorktree(config.AgentID(id), wt)
		}
	}
	if len(ns.PromptHist) > 0 {
		m.session.PromptHist = append([]string{}, ns.PromptHist...)
	}
	if ns.ActiveID != "" {
		for i, a := range m.agents {
			if a.ID == ns.ActiveID {
				m.selected = i
				m.activeID = a.ID
				break
			}
		}
	}
	m.session.LastAgentID = m.activeID
	_ = config.SaveSession(m.session)
	m.statusMsg = fmt.Sprintf("loaded session %q", pick)
	m.refreshViewport()
	return m, nil
}

func (m Model) resendLastPrompt() (tea.Model, tea.Cmd) {
	if len(m.session.PromptHist) == 0 {
		m.statusMsg = "no prompt history"
		return m, nil
	}
	text := m.session.PromptHist[len(m.session.PromptHist)-1]
	id := m.agents[m.selected].ID
	p := m.manager.Get(id)
	if p == nil || !p.IsLive() {
		m.statusMsg = "start the agent before resending"
		return m, nil
	}
	if err := m.manager.Send(id, text); err != nil {
		m.statusMsg = fmt.Sprintf("resend failed: %v", err)
	} else {
		m.statusMsg = fmt.Sprintf("resent to %s (%d chars)", p.Name, len(text))
		m.refreshViewport()
	}
	return m, nil
}

func (m Model) exportOutput() (tea.Model, tea.Cmd) {
	id := m.agents[m.selected].ID
	p := m.manager.Get(id)
	if p == nil {
		m.statusMsg = "nothing to export"
		return m, nil
	}
	lines := p.SnapshotOutput()
	if len(lines) == 0 {
		m.statusMsg = "nothing to export"
		return m, nil
	}
	dir := m.manager.Cwd()
	name := fmt.Sprintf("fleet-%s-%s.log", id, time.Now().Format("150405"))
	path := filepath.Join(dir, name)
	header := fmt.Sprintf("# %s · %s · %s\n# model: %s\n\n", p.Name, p.Status, m.manager.EffectiveCwd(id), p.Model)
	body := header + strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		m.statusMsg = fmt.Sprintf("export failed: %v", err)
	} else {
		m.statusMsg = fmt.Sprintf("exported %d lines → %s", len(lines), name)
	}
	return m, nil
}

func (m Model) openModelPicker() (tea.Model, tea.Cmd) {
	id := m.activeID
	if len(m.agents) > 0 {
		id = m.agents[m.selected].ID
	}
	// Instant defaults so the UI never waits on CLI discovery
	defaults := append(append([]string{}, m.manager.ModelsForDefaultsOnly(id)...), "custom…")
	m.modelList = defaults
	m.modelIdx = 0
	cur := m.manager.GetModel(id)
	for i, s := range defaults {
		if s == cur {
			m.modelIdx = i
			break
		}
	}
	m.modelMode = true
	m.statusMsg = fmt.Sprintf("model · %s — loading live list…", id)
	agentID := id
	return m, func() tea.Msg {
		list := m.manager.ModelsFor(agentID)
		src := "defaults"
		if len(list) > len(config.DefaultModels(agentID)) {
			src = "live+defaults"
		}
		return modelsMsg{id: agentID, list: list, source: src}
	}
}

func (m Model) updateModelPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.FocusOut), key.Matches(msg, m.keys.Quit):
		m.modelMode = false
		m.statusMsg = "model select cancelled"
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.modelIdx > 0 {
			m.modelIdx--
		} else if len(m.modelList) > 0 {
			m.modelIdx = len(m.modelList) - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.modelIdx < len(m.modelList)-1 {
			m.modelIdx++
		} else {
			m.modelIdx = 0
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if len(m.modelList) == 0 {
			m.modelMode = false
			return m, nil
		}
		choice := m.modelList[m.modelIdx]
		id := m.agents[m.selected].ID
		if choice == "custom…" {
			m.modelMode = false
			m.inputFocus = true
			m.nameMode = false
			m.pathMode = false
			m.input.SetValue(m.manager.GetModel(id))
			m.input.Placeholder = "Type model id and press Enter…"
			m.input.Focus()
			m.statusMsg = "custom model — Enter to set, Esc cancel"
			// mark pending custom model via status convention handled in updateInput
			m.fanMode = false
			m.modelMode = false
			// reuse nameMode=false; use a dedicated flag
			m.modelCustom = true
			m.layout()
			return m, textarea.Blink
		}
		if err := m.manager.ApplyModel(id, choice); err != nil {
			m.statusMsg = fmt.Sprintf("set %s, live switch: %v", choice, err)
		} else {
			st := m.manager.Get(id)
			if st != nil && st.IsLive() {
				m.statusMsg = fmt.Sprintf("%s → %s (sent /model to live session)", m.agents[m.selected].Name, choice)
			} else {
				m.statusMsg = fmt.Sprintf("%s → %s (will use on next start)", m.agents[m.selected].Name, choice)
			}
		}
		m.modelMode = false
		m.agents = m.manager.List()
		m.refreshViewport()
		return m, nil
	}
	return m, nil
}

func (m Model) updateTemplatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.FocusOut):
		m.tmplMode = false
		m.statusMsg = "templates closed"
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.tmplIdx > 0 {
			m.tmplIdx--
		} else if len(promptTemplates) > 0 {
			m.tmplIdx = len(promptTemplates) - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.tmplIdx < len(promptTemplates)-1 {
			m.tmplIdx++
		} else {
			m.tmplIdx = 0
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		tmpl := promptTemplates[m.tmplIdx]
		m.tmplMode = false
		m.inputFocus = true
		m.input.SetValue("/" + tmpl.ID)
		m.input.Focus()
		m.layout()
		m.statusMsg = fmt.Sprintf("template /%s — edit or Enter to send", tmpl.ID)
		return m, textarea.Blink
	case msg.String() >= "1" && msg.String() <= "4":
		tmpl := promptTemplates[m.tmplIdx]
		if m.session.Pins == nil {
			m.session.Pins = map[string]string{}
		}
		m.session.Pins[msg.String()] = tmpl.ID
		_ = config.SaveSession(m.session)
		m.statusMsg = fmt.Sprintf("pinned /%s to slot %s", tmpl.ID, msg.String())
		return m, nil
	}
	return m, nil
}

func (m Model) ensureWorktree() (tea.Model, tea.Cmd) {
	cwd := m.manager.Cwd()
	if !gitutil.IsRepo(cwd) {
		m.statusMsg = "not a git repo — worktrees need a repository"
		return m, nil
	}
	root, err := gitutil.Root(cwd)
	if err != nil {
		m.statusMsg = fmt.Sprintf("git root: %v", err)
		return m, nil
	}
	id := m.agents[m.selected].ID
	wt, err := gitutil.EnsureWorktree(root, string(id))
	if err != nil {
		m.statusMsg = fmt.Sprintf("worktree failed: %v", err)
		return m, nil
	}
	m.manager.SetWorktree(id, wt)
	// persist immediately
	if m.session.Worktrees == nil {
		m.session.Worktrees = map[string]string{}
	}
	m.session.Worktrees[string(id)] = wt
	_ = config.SaveSession(m.session)
	m.statusMsg = fmt.Sprintf("worktree ready: %s", gitutil.ShortName(wt))
	m.agents = m.manager.List()
	return m, nil
}
