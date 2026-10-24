package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/feralxr/agent-fleet/internal/config"
	"github.com/feralxr/agent-fleet/internal/gitutil"
	"github.com/feralxr/agent-fleet/internal/process"
)

const (
	tickInterval = 200 * time.Millisecond
)

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Quit     key.Binding
	Help     key.Binding
	Start    key.Binding
	Stop     key.Binding
	StopAll  key.Binding
	Restart  key.Binding
	Clear    key.Binding
	FocusIn  key.Binding
	FocusOut key.Binding
	Copy     key.Binding
	Cwd      key.Binding
	Split    key.Binding
	Worktree key.Binding
	Fan      key.Binding
	Jump     key.Binding
	Diff     key.Binding
	Reload   key.Binding
	SaveName key.Binding
	LoadName key.Binding
	HistUp   key.Binding
	HistDown key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "prev agent"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "next agent"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send / confirm"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start agent"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop agent"),
		),
		StopAll: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "stop all agents"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart agent"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear output"),
		),
		FocusIn: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "focus input"),
		),
		FocusOut: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel / unfocus"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy output"),
		),
		Cwd: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "change directory"),
		),
		Split: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "toggle split view"),
		),
		Worktree: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "create worktree"),
		),
		Fan: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fan prompt to all"),
		),
		Jump: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "jump to needs attention"),
		),
		Diff: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "toggle git diff"),
		),
		Reload: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "reload fleet.toml"),
		),
		SaveName: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "save named session"),
		),
		LoadName: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "load named session"),
		),
		HistUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "prev prompt (input)"),
		),
		HistDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next prompt (input)"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Start, k.Stop, k.Jump, k.Diff, k.Worktree, k.Fan, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Start, k.Stop, k.StopAll, k.Restart},
		{k.Jump, k.Diff, k.Worktree, k.Fan, k.Cwd, k.Split},
		{k.SaveName, k.LoadName, k.Reload, k.Copy, k.Clear},
		{k.FocusIn, k.FocusOut, k.Enter, k.Help, k.Quit},
	}
}

// Model is the main Bubble Tea model.
type Model struct {
	manager      *process.Manager
	agents       []*process.AgentProcess
	selected     int
	activeID     config.AgentID
	viewport     viewport.Model
	viewport2    viewport.Model // secondary pane for split view
	input        textarea.Model
	help         help.Model
	keys         keyMap
	renderer     *glamour.TermRenderer
	session      config.Session
	width        int
	height       int
	showHelp     bool
	inputFocus   bool
	pathMode     bool // true when editing working directory
	fanMode      bool // next prompt fans to all running agents
	showDiff     bool // show git diff --stat instead of agent output
	split        bool // side-by-side output of two agents
	nameMode     bool // saving a named session
	histIdx      int  // -1 = not browsing history
	statusMsg    string
	lastTick     time.Time
	ready        bool
	lastContent  string
	lastContent2 string
}

// NewModel creates the initial UI model.
func NewModel() Model {
	session := config.LoadSession()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	// Prefer last used working directory if it still exists.
	if session.LastCwd != "" {
		if st, err := os.Stat(session.LastCwd); err == nil && st.IsDir() {
			cwd = session.LastCwd
		}
	}

	mgr := process.NewManager(cwd, &session)
	agents := mgr.List()

	selected := 0
	activeID := agents[0].ID
	if session.LastAgentID != "" {
		for i, a := range agents {
			if a.ID == session.LastAgentID {
				selected = i
				activeID = a.ID
				break
			}
		}
	}

	ta := textarea.New()
	ta.Placeholder = "Type a prompt and press Enter to send to the active agent…"
	ta.CharLimit = 4000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Focus()

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to Agent Fleet\n\nSelect an agent on the left and press `s` to start it.")
	vp2 := viewport.New(40, 20)
	vp2.SetContent("")

	h := help.New()

	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	m := Model{
		manager:    mgr,
		agents:     agents,
		selected:   selected,
		activeID:   activeID,
		viewport:   vp,
		viewport2:  vp2,
		input:      ta,
		help:       h,
		keys:       defaultKeyMap(),
		renderer:   r,
		session:    session,
		inputFocus: false,
		pathMode:   false,
		fanMode:    false,
		showDiff:   false,
		split:      false,
		nameMode:   false,
		histIdx:    -1,
		statusMsg:  "Ready",
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tickCmd(),
	)
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layout()
		// Keep PTY size roughly aligned with the output pane
		rows := uint16(m.viewport.Height)
		cols := uint16(m.viewport.Width)
		if rows > 0 && cols > 0 {
			m.manager.SetPTYSize(m.activeID, rows, cols)
		}
		return m, nil

	case tickMsg:
		m.agents = m.manager.List()
		m.refreshViewport()
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if m.inputFocus {
			return m.updateInput(msg)
		}
		return m.updateNormal(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		// stop all running agents on quit
		for _, a := range m.agents {
			_ = m.manager.Stop(a.ID)
		}
		// persist session
		m.session.LastCwd = m.manager.Cwd()
		m.session.LastAgentID = m.activeID
		m.session.Worktrees = map[string]string{}
		for _, a := range m.agents {
			if wt := m.manager.Worktree(a.ID); wt != "" {
				m.session.Worktrees[string(a.ID)] = wt
			}
		}
		_ = config.SaveSession(m.session)
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.selected > 0 {
			m.selected--
			m.activeID = m.agents[m.selected].ID
			m.refreshViewport()
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.selected < len(m.agents)-1 {
			m.selected++
			m.activeID = m.agents[m.selected].ID
			m.refreshViewport()
		}
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
		m.lastContent = "" // force redraw
		m.lastContent2 = ""
		if m.showDiff {
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

	case key.Matches(msg, m.keys.FocusIn):
		if m.pathMode {
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
		m.input.Placeholder = "Type a prompt and press Enter to send to the active agent…"
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
			m.pushPrompt(text)
			n := 0
			var lastErr error
			for _, a := range m.agents {
				if a.Status == process.StatusRunning || a.Status == process.StatusWaiting {
					if err := m.manager.Send(a.ID, text); err != nil {
						lastErr = err
					} else {
						n++
					}
				}
			}
			m.fanMode = false
			m.histIdx = -1
			m.inputFocus = false
			m.input.Blur()
			m.input.Reset()
			m.input.Placeholder = "Type a prompt and press Enter to send to the active agent…"
			if n == 0 {
				m.statusMsg = "fan-out: no running agents"
			} else if lastErr != nil {
				m.statusMsg = fmt.Sprintf("fan-out: sent to %d (errors: %v)", n, lastErr)
			} else {
				m.statusMsg = fmt.Sprintf("fan-out: sent to %d agents", n)
			}
			m.refreshViewport()
			return m, nil
		}

		m.pushPrompt(text)
		err := m.manager.Send(m.activeID, text)
		if err != nil {
			m.statusMsg = fmt.Sprintf("send failed: %v", err)
		} else {
			m.statusMsg = "prompt sent"
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

func (m *Model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	leftWidth := 26
	if m.width < 90 {
		leftWidth = 22
	}
	if m.width < 70 {
		leftWidth = 18
	}

	chrome := 9
	contentHeight := m.height - chrome
	if contentHeight < 6 {
		contentHeight = 6
	}

	avail := m.width - leftWidth - 1
	if avail < 20 {
		avail = 20
	}

	if m.split {
		half := avail / 2
		m.viewport.Width = half
		m.viewport.Height = contentHeight
		m.viewport2.Width = avail - half
		m.viewport2.Height = contentHeight
	} else {
		m.viewport.Width = avail
		m.viewport.Height = contentHeight
		m.viewport2.Width = 0
		m.viewport2.Height = contentHeight
	}

	m.input.SetWidth(m.width - 2)
	m.input.SetHeight(3)

	wrap := m.viewport.Width - 4
	if wrap < 20 {
		wrap = 20
	}
	if m.renderer != nil {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(wrap),
		)
		if err == nil {
			m.renderer = r
			m.lastContent = ""
			m.lastContent2 = ""
		}
	}
}

func (m *Model) refreshViewport() {
	m.renderAgentInto(&m.viewport, &m.lastContent, m.activeID)

	if m.split {
		// Secondary pane: previous agent in list, or next if at top.
		secIdx := m.selected - 1
		if secIdx < 0 {
			secIdx = m.selected + 1
		}
		if secIdx >= 0 && secIdx < len(m.agents) {
			m.renderAgentInto(&m.viewport2, &m.lastContent2, m.agents[secIdx].ID)
		} else {
			m.viewport2.SetContent("(no second agent)")
		}
	}
}

func (m *Model) renderAgentInto(vp *viewport.Model, last *string, id config.AgentID) {
	p := m.manager.Get(id)
	if p == nil {
		content := "(no agent selected)"
		if content != *last {
			*last = content
			vp.SetContent(content)
		}
		return
	}

	if m.showDiff {
		dir := m.manager.EffectiveCwd(id)
		stat := gitutil.DiffStat(dir)
		var content string
		if stat == "" {
			content = fmt.Sprintf("# %s — git diff\n\n`%s`\n\n_Working tree clean (or not a git repo)._\n", p.Name, dir)
		} else {
			content = fmt.Sprintf("# %s — git diff\n\n`%s`\n\n```\n%s\n```\n", p.Name, dir, stat)
		}
		content = m.renderMarkdown(content)
		if content != *last {
			*last = content
			vp.SetContent(content)
			vp.GotoTop()
		}
		return
	}

	lines := p.SnapshotOutput()
	if len(lines) == 0 {
		dir := m.manager.EffectiveCwd(id)
		mode := "pipes"
		if p.UsePTY {
			mode = "pty"
		}
		msg := fmt.Sprintf("# %s\n\n**Status:** `%s`  ·  **Mode:** `%s`\n\n**Path:** `%s`\n",
			p.Name, p.Status, mode, dir)
		switch p.Status {
		case process.StatusMissing:
			msg += fmt.Sprintf("\nBinary **not found** on `$PATH`: `%s`\n\n"+
				"Fix options:\n"+
				"1. Install the CLI and ensure it is on `$PATH`\n"+
				"2. Set env `FLEET_%s_CMD` to the full binary path\n"+
				"3. Add a custom entry in `fleet.toml` and press `ctrl+r`\n",
				p.CmdPath, strings.ToUpper(strings.ReplaceAll(string(p.ID), "-", "_")))
		case process.StatusError:
			err := p.LastError()
			if err == "" {
				err = "unknown error"
			}
			msg += fmt.Sprintf("\nLast error: `%s`\n\nPress **r** to restart or **s** to start again.\n", err)
		case process.StatusStopped:
			msg += "\nAgent stopped. Press **s** to start again, or **r** to restart.\n"
		default:
			msg += "\nPress **s** to start this agent.\n\n" +
				"| Key | Action |\n|-----|--------|\n" +
				"| `s` | Start |\n| `w` | Create git worktree |\n| `i` | Focus prompt |\n| `g` | Diff view |\n| `?` | Full help |\n"
		}
		content := m.renderMarkdown(msg)
		if content != *last {
			*last = content
			vp.SetContent(content)
			vp.GotoTop()
		}
		return
	}

	// Stick-to-bottom: only auto-scroll when user was already at bottom
	atBottom := vp.AtBottom()

	var b strings.Builder
	// Lightweight session header
	if u := process.FormatUptime(p.Uptime()); u != "" {
		b.WriteString(subtleStyle.Render(fmt.Sprintf("── %s · up %s · %d lines ──", p.Name, u, len(lines))) + "\n")
	}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "[fleet]"):
			b.WriteString(fleetLineStyle.Render(line) + "\n")
		case strings.HasPrefix(line, "› "):
			b.WriteString(promptLineStyle.Render(line) + "\n")
		default:
			b.WriteString(line + "\n")
		}
	}
	content := b.String()
	// Markdown only for modest buffers (keeps streaming fast)
	if len(lines) < 300 && m.renderer != nil {
		if rendered, err := m.renderer.Render(content); err == nil {
			content = rendered
		}
	}
	if content != *last {
		*last = content
		vp.SetContent(content)
		if atBottom {
			vp.GotoBottom()
		}
	}
}

func (m *Model) setViewportContent(content string) {
	if content == m.lastContent {
		return // skip identical re-render
	}
	m.lastContent = content
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m *Model) renderMarkdown(src string) string {
	if m.renderer == nil {
		return src
	}
	out, err := m.renderer.Render(src)
	if err != nil {
		return src
	}
	return out
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Loading Agent Fleet…"
	}

	title := titleStyle.Render(" Agent Fleet ")
	var badges []string
	if m.split {
		badges = append(badges, "split")
	}
	if m.pathMode {
		badges = append(badges, "cwd")
	}
	if m.showDiff {
		badges = append(badges, "diff")
	}
	if m.fanMode {
		badges = append(badges, "fan")
	}
	if m.nameMode {
		badges = append(badges, "save")
	}
	mode := ""
	if len(badges) > 0 {
		mode = subtleStyle.Render("  " + strings.Join(badges, " · "))
	}
	// Active agent chip
	chip := ""
	if p := m.manager.Get(m.activeID); p != nil {
		chip = subtleStyle.Render(fmt.Sprintf("  %s [%s]", p.Name, p.Status))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, mode, chip)

	leftWidth := 26
	if m.width < 90 {
		leftWidth = 22
	}
	if m.width < 70 {
		leftWidth = 18
	}
	rightWidth := m.width - leftWidth - 1
	if rightWidth < 20 {
		rightWidth = 20
	}

	left := m.renderAgentList()
	leftPane := leftStyle.Width(leftWidth).Height(m.viewport.Height).Render(left)

	var rightPane string
	if m.split {
		half := rightWidth / 2
		p1 := rightStyle.Width(half).Height(m.viewport.Height).Render(m.viewport.View())
		p2 := rightStyle.Width(rightWidth - half).Height(m.viewport.Height).Render(m.viewport2.View())
		rightPane = lipgloss.JoinHorizontal(lipgloss.Top, p1, p2)
	} else {
		rightPane = rightStyle.Width(rightWidth).Height(m.viewport.Height).Render(m.viewport.View())
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	inputBorderColor := lipgloss.Color("#374151")
	if m.inputFocus || m.pathMode {
		inputBorderColor = lipgloss.Color("#7C3AED")
	}
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorderColor).
		Padding(0, 1).
		Width(m.width - 2).
		Render(m.input.View())

	cwd := m.manager.Cwd()
	if len(cwd) > 40 {
		cwd = "…" + cwd[len(cwd)-37:]
	}
	activeName := ""
	if p := m.manager.Get(m.activeID); p != nil {
		activeName = p.Name
	}
	attn := m.manager.AttentionCount()
	runn := m.manager.RunningCount()
	badge := fmt.Sprintf("%d run", runn)
	if attn > 0 {
		badge += fmt.Sprintf(" · %d need you", attn)
	}
	statusLeft := fmt.Sprintf(" %s", m.statusMsg)
	statusRight := fmt.Sprintf("%s  ·  %s  ·  %s  ·  %s ", badge, activeName, cwd, time.Now().Format("15:04:05"))
	gap := m.width - lipgloss.Width(statusLeft) - lipgloss.Width(statusRight)
	if gap < 1 {
		gap = 1
	}
	status := statusStyle.Width(m.width).Render(
		statusLeft + strings.Repeat(" ", gap) + statusRight,
	)

	var helpView string
	if m.showHelp {
		helpView = m.help.View(m.keys)
	} else {
		helpView = subtleStyle.Render(" ? help · space · g diff · S/L sessions · ↑↓ history · s start · w worktree · f fan · q quit")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		body,
		"",
		inputBox,
		status,
		helpView,
	)
}

func (m Model) renderAgentList() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("AGENTS") + "\n")

	for i, a := range m.agents {
		selected := i == m.selected

		var statusDot string
		switch a.Status {
		case process.StatusRunning:
			statusDot = statusRun.Render("●")
		case process.StatusWaiting:
			statusDot = statusWait.Render("◐")
		case process.StatusError:
			statusDot = statusErr.Render("●")
		case process.StatusMissing:
			statusDot = statusMiss.Render("●")
		default:
			statusDot = statusIdle.Render("●")
		}

		name := a.Name
		if selected {
			name = selectedStyle.Render(a.Name)
		}

		// Active row gets a left bar + background feel
		prefix := "  "
		if selected {
			prefix = activeBar.Render("▌") + " "
		}

		line := fmt.Sprintf("%s%s %s", prefix, statusDot, name)
		if selected {
			line = selectedRow.Render(line)
		}
		b.WriteString(line + "\n")

		// Secondary status text
		st := string(a.Status)
		if a.Status == process.StatusMissing {
			st = "not found"
			if a.CmdPath != "" {
				st = "missing: " + a.ShortPath()
			}
		} else if a.Status == process.StatusWaiting {
			st = "needs you"
			if a.WorktreePath != "" {
				st = "needs you · wt"
			}
		} else if a.WorktreePath != "" {
			st = "wt:" + gitutil.ShortName(a.WorktreePath)
			if a.Status == process.StatusRunning || a.Status == process.StatusWaiting {
				if u := process.FormatUptime(a.Uptime()); u != "" {
					st += " · " + u
				}
			}
		} else if a.Status == process.StatusRunning || a.Status == process.StatusWaiting {
			if u := process.FormatUptime(a.Uptime()); u != "" {
				st = u
			} else {
				st = string(a.Status)
			}
		} else if a.Available && a.CmdPath != "" && a.Status == process.StatusIdle {
			st = a.ShortPath()
			if a.UsePTY {
				st += " · pty"
			}
		}
		// Append one-line diff summary when dirty
		if dir := m.manager.EffectiveCwd(a.ID); dir != "" {
			if d := gitutil.DiffShortOneLiner(dir); d != "" && d != "clean" {
				if st != "" {
					st += " · "
				}
				st += d
			}
		}
		sec := subtleStyle.Render(fmt.Sprintf("    %s", st))
		b.WriteString(sec + "\n")
	}

	return b.String()
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E9D5FF"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A78BFA")).
			MarginBottom(1)

	leftStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#27272A")).
			Padding(0, 1, 0, 0)

	rightStyle = lipgloss.NewStyle().
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A1A1AA")).
			Background(lipgloss.Color("#18181B"))

	statusIdle = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
	statusRun  = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399"))
	statusWait = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
	statusErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	statusMiss = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))

	fleetLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B")).Italic(true)
	promptLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD")).Bold(true)

	activeBar   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	selectedRow = lipgloss.NewStyle().Background(lipgloss.Color("#1E1B2E"))
)
