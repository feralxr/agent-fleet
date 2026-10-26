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
	tickInterval = 500 * time.Millisecond
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
	ModelSel key.Binding
	PgUp     key.Binding
	PgDown   key.Binding
	HalfUp   key.Binding
	HalfDown key.Binding
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
		ModelSel: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "select model"),
		),
		PgUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+b"),
			key.WithHelp("pgup", "scroll up"),
		),
		PgDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("pgdown", "scroll down"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		HalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
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
		{k.PgUp, k.PgDown, k.HalfUp, k.HalfDown, k.Copy, k.Clear},
		{k.SaveName, k.LoadName, k.Reload, k.FocusIn, k.Help, k.Quit},
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
	modelMode    bool // model picker open
	modelCustom  bool // typing a custom model id
	modelList    []string
	modelIdx     int
	histIdx      int  // -1 = not browsing history
	statusMsg    string
	lastTick     time.Time
	ready        bool
	lastContent  string
	lastContent2 string
	outFP        string // fingerprint of active agent output
	outFP2       string
	diffCache    map[string]diffCacheEntry
	diffCacheAt  time.Time
}

type diffCacheEntry struct {
	short string
	full  string
	at    time.Time
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
	ta.Placeholder = "i  focus  ·  enter  send  ·  esc  cancel"
	ta.CharLimit = 4000
	ta.SetHeight(2)
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#F4F4F5"))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#1E1033"))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	ta.Blur()

	vp := viewport.New(80, 20)
	vp.SetContent("agent fleet\n\n  j/k or 1-9   select\n  s            start\n  i            prompt\n  space        jump to wait\n  ?            help\n")
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
		diffCache:  make(map[string]diffCacheEntry),
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
		// At most one git status refresh per tick (never from View)
		m.refreshDiffCacheLazy()
		// Only re-render panes when agent output actually changed
		if m.outputChanged(m.activeID, &m.outFP) || (m.split && m.outputChanged(m.secondAgentID(), &m.outFP2)) {
			m.refreshViewport()
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if m.inputFocus {
			return m.updateInput(msg)
		}
		return m.updateNormal(msg)

	case tea.MouseMsg:
		if m.inputFocus {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			m.viewport.LineDown(3)
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.modelMode {
		return m.updateModelPicker(msg)
	}
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
		m.session.Models = map[string]string{}
		for _, a := range m.agents {
			if wt := m.manager.Worktree(a.ID); wt != "" {
				m.session.Worktrees[string(a.ID)] = wt
			}
			if mod := m.manager.GetModel(a.ID); mod != "" {
				m.session.Models[string(a.ID)] = mod
			}
		}
		_ = config.SaveSession(m.session)
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
		// Jump to agent by number (1-indexed)
		n := int(msg.String()[0] - '0')
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

		if m.modelCustom {
			id := m.agents[m.selected].ID
			if err := m.manager.ApplyModel(id, text); err != nil {
				m.statusMsg = fmt.Sprintf("set %s, live switch: %v", text, err)
			} else {
				st := m.manager.Get(id)
				if st != nil && (st.Status == process.StatusRunning || st.Status == process.StatusWaiting) {
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

func (m *Model) outputChanged(id config.AgentID, fp *string) bool {
	p := m.manager.Get(id)
	if p == nil {
		if *fp != "" {
			*fp = ""
			return true
		}
		return false
	}
	next := p.OutputFingerprint()
	if next != *fp {
		*fp = next
		return true
	}
	return false
}

func (m Model) secondAgentID() config.AgentID {
	if len(m.agents) < 2 {
		return ""
	}
	secIdx := m.selected + 1
	if secIdx >= len(m.agents) {
		secIdx = 0
	}
	return m.agents[secIdx].ID
}

const diffCacheTTL = 3 * time.Second

// cachedDiffShort is View-safe: never shells out to git.
func (m *Model) cachedDiffShort(dir string) string {
	if dir == "" || m.diffCache == nil {
		return ""
	}
	if e, ok := m.diffCache[dir]; ok {
		return e.short
	}
	return ""
}

// cachedDiffFull returns a cached full stat; refreshes if missing/stale (only call from key handlers / tick, not View for many agents).
func (m *Model) cachedDiffFull(dir string) string {
	if dir == "" {
		return ""
	}
	if e, ok := m.diffCache[dir]; ok && e.full != "" && time.Since(e.at) < diffCacheTTL {
		return e.full
	}
	m.refreshDiffCache(dir)
	return m.diffCache[dir].full
}

func (m *Model) refreshDiffCache(dir string) {
	if dir == "" {
		return
	}
	if m.diffCache == nil {
		m.diffCache = make(map[string]diffCacheEntry)
	}
	short := gitutil.DiffShortOneLiner(dir)
	full := gitutil.DiffStat(dir)
	m.diffCache[dir] = diffCacheEntry{short: short, full: full, at: time.Now()}
}

// refreshDiffCacheLazy updates at most one stale path per tick.
func (m *Model) refreshDiffCacheLazy() {
	for _, a := range m.agents {
		dir := m.manager.EffectiveCwd(a.ID)
		if dir == "" {
			continue
		}
		e, ok := m.diffCache[dir]
		if !ok || time.Since(e.at) >= diffCacheTTL {
			m.refreshDiffCache(dir)
			return // one git batch per tick max
		}
	}
}


func (m Model) openModelPicker() (tea.Model, tea.Cmd) {
	id := m.activeID
	if len(m.agents) > 0 {
		id = m.agents[m.selected].ID
	}
	m.statusMsg = fmt.Sprintf("loading models for %s…", id)
	list := m.manager.ModelsFor(id)
	// Always allow custom entry as last item
	list = append(append([]string{}, list...), "custom…")
	m.modelList = list
	m.modelIdx = 0
	cur := m.manager.GetModel(id)
	for i, s := range list {
		if s == cur {
			m.modelIdx = i
			break
		}
	}
	m.modelMode = true
	src := "defaults"
	if len(list) > len(config.DefaultModels(id))+1 {
		src = "live+defaults"
	}
	m.statusMsg = fmt.Sprintf("model · %s (%d, %s) — j/k · enter · esc", id, len(list)-1, src)
	return m, nil
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
			if st != nil && (st.Status == process.StatusRunning || st.Status == process.StatusWaiting) {
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

func (m Model) sidebarWidth() int {
	if m.width < 80 {
		return 20
	}
	if m.width < 100 {
		return 24
	}
	return 28
}

func (m *Model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	leftWidth := m.sidebarWidth()

	// header(1) + body headers(1) + input(~3) + status(1) + help(1) + borders
	chrome := 10
	if m.inputFocus || m.pathMode || m.nameMode || m.fanMode {
		chrome = 11
	}
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

	m.input.SetWidth(max(m.width-4, 20))
	if m.inputFocus || m.pathMode || m.nameMode || m.fanMode {
		m.input.SetHeight(3)
	} else {
		m.input.SetHeight(2)
	}

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
		stat := m.cachedDiffFull(dir)
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
		var b strings.Builder
		b.WriteString(selectedStyle.Render(p.Name) + "\n")
		b.WriteString(subtleStyle.Render(fmt.Sprintf("%s  ·  %s  ·  %s", p.Status, mode, dir)) + "\n\n")
		switch p.Status {
		case process.StatusMissing:
			b.WriteString(statusMiss.Render("binary not found") + "\n")
			b.WriteString(subtleStyle.Render(p.CmdPath) + "\n\n")
			b.WriteString(subtleStyle.Render("install the CLI, set FLEET_"+strings.ToUpper(strings.ReplaceAll(string(p.ID), "-", "_"))+"_CMD,") + "\n")
			b.WriteString(subtleStyle.Render("or add it in fleet.toml and press ctrl+r") + "\n")
		case process.StatusError:
			err := p.LastError()
			if err == "" {
				err = "unknown error"
			}
			b.WriteString(statusErr.Render("error") + "  " + subtleStyle.Render(err) + "\n\n")
			b.WriteString(subtleStyle.Render("r  restart   s  start") + "\n")
		case process.StatusStopped:
			b.WriteString(subtleStyle.Render("stopped — press s to start") + "\n")
		default:
			b.WriteString(subtleStyle.Render("s  start     w  worktree") + "\n")
			b.WriteString(subtleStyle.Render("i  prompt    g  diff") + "\n")
			b.WriteString(subtleStyle.Render("?  help") + "\n")
		}
		content := b.String()
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
	meta := fmt.Sprintf("%s", p.Name)
	if u := process.FormatUptime(p.Uptime()); u != "" {
		meta += "  ·  up " + u
	}
	meta += fmt.Sprintf("  ·  %d lines", len(lines))
	b.WriteString(subtleStyle.Render(meta) + "\n")
	b.WriteString(subtleStyle.Render(strings.Repeat("─", min(48, max(m.viewport.Width-2, 8)))) + "\n")
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
	// No Glamour on live streams — ANSI/markdown render is too expensive for 500ms ticks.
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

	title := titleStyle.Render(" ✦ Agent Fleet ")
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
	if m.modelMode || m.modelCustom {
		badges = append(badges, "model")
	}
	if m.inputFocus && !m.pathMode && !m.nameMode && !m.fanMode {
		badges = append(badges, "input")
	}
	mode := ""
	if len(badges) > 0 {
		mode = badgeStyle.Render(" " + strings.Join(badges, " · ") + " ")
	}
	chip := ""
	if p := m.manager.Get(m.activeID); p != nil {
		st := statusIdle
		switch p.Status {
		case process.StatusRunning:
			st = statusRun
		case process.StatusWaiting:
			st = statusWait
		case process.StatusError:
			st = statusErr
		case process.StatusMissing:
			st = statusMiss
		}
		chip = " " + st.Render("●") + subtleStyle.Render(" "+p.Name)
	}
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, " ", mode, chip)

	leftWidth := m.sidebarWidth()
	rightWidth := m.width - leftWidth - 1
	if rightWidth < 20 {
		rightWidth = 20
	}

	left := m.renderAgentList()
	leftPane := leftStyle.Width(leftWidth).Height(m.viewport.Height + 1).Render(left)

	outTitle := "OUTPUT"
	if m.showDiff {
		outTitle = "DIFF"
	}
	if p := m.manager.Get(m.activeID); p != nil {
		outTitle += " · " + p.Name
	}
	// Scroll position (built-in viewport percent)
	if m.viewport.TotalLineCount() > m.viewport.Height {
		if m.viewport.AtBottom() {
			outTitle += " · end"
		} else if m.viewport.AtTop() {
			outTitle += " · top"
		} else {
			outTitle += fmt.Sprintf(" · %d%%", int(m.viewport.ScrollPercent()*100))
		}
	}

	var rightPane string
	if m.split {
		half := rightWidth / 2
		secName := "…"
		if id := m.secondAgentID(); id != "" {
			if p := m.manager.Get(id); p != nil {
				secName = p.Name
			}
		}
		actName := string(m.activeID)
		if p := m.manager.Get(m.activeID); p != nil {
			actName = p.Name
		}
		h1 := sectionStyle.Width(max(half-2, 8)).Render(" " + actName + " ")
		h2 := sectionStyle.Width(max(rightWidth-half-2, 8)).Render(" " + secName + " ")
		p1 := rightStyle.Width(half).Height(m.viewport.Height).Render(m.viewport.View())
		p2 := rightStyle.Width(rightWidth - half).Height(m.viewport.Height).Render(m.viewport2.View())
		rightPane = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left, h1, p1),
			lipgloss.JoinVertical(lipgloss.Left, h2, p2),
		)
	} else {
		h := sectionStyle.Width(max(rightWidth-2, 8)).Render(" " + outTitle + " ")
		pane := rightStyle.Width(rightWidth).Height(m.viewport.Height).Render(m.viewport.View())
		rightPane = lipgloss.JoinVertical(lipgloss.Left, h, pane)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	inputBorderColor := lipgloss.Color("#3F3F46")
	inputLabel := subtleStyle.Render(" prompt ")
	switch {
	case m.fanMode:
		inputBorderColor = lipgloss.Color("#38BDF8")
		inputLabel = statusWait.Render(" fan-out ")
	case m.pathMode:
		inputBorderColor = lipgloss.Color("#FBBF24")
		inputLabel = statusMiss.Render(" cwd ")
	case m.nameMode:
		inputBorderColor = lipgloss.Color("#A78BFA")
		inputLabel = selectedStyle.Render(" session ")
	case m.inputFocus:
		inputBorderColor = lipgloss.Color("#8B5CF6")
		inputLabel = selectedStyle.Render(" input ")
	}
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorderColor).
		Padding(0, 1).
		Width(m.width - 2).
		Render(inputLabel + m.input.View())

	cwd := m.manager.Cwd()
	if len(cwd) > 36 {
		cwd = "…" + cwd[len(cwd)-33:]
	}
	attn := m.manager.AttentionCount()
	runn := m.manager.RunningCount()
	var runBadge string
	if attn > 0 {
		runBadge = statusWait.Render(fmt.Sprintf("%d run · %d need you", runn, attn))
	} else {
		runBadge = subtleStyle.Render(fmt.Sprintf("%d running", runn))
	}
	msgStyle := subtleStyle
	sm := strings.ToLower(m.statusMsg)
	switch {
	case strings.Contains(sm, "fail") || strings.Contains(sm, "error") || strings.Contains(sm, "not found"):
		msgStyle = statusErr
	case strings.Contains(sm, "started") || strings.Contains(sm, "sent") || strings.Contains(sm, "saved") || strings.Contains(sm, "ready") || strings.Contains(sm, "copied"):
		msgStyle = statusRun
	case strings.Contains(sm, "need") || strings.Contains(sm, "fan") || strings.Contains(sm, "jump"):
		msgStyle = statusWait
	}
	statusLeft := " " + msgStyle.Render(m.statusMsg)
	if lipgloss.Width(statusLeft) > 52 {
		// strip styles for length check is hard; keep raw truncate on original
		raw := " " + m.statusMsg
		if len(raw) > 48 {
			statusLeft = msgStyle.Render(raw[:45] + "…")
		}
	}
	statusRight := fmt.Sprintf("%s  %s  %s ", runBadge, subtleStyle.Render(cwd), subtleStyle.Render(time.Now().Format("15:04")))
	gap := m.width - lipgloss.Width(statusLeft) - lipgloss.Width(statusRight)
	if gap < 1 {
		gap = 1
	}
	status := statusStyle.Width(m.width).Render(statusLeft + strings.Repeat(" ", gap) + statusRight)

	var helpView string
	if m.showHelp {
		helpView = m.help.View(m.keys)
	} else {
		helpView = subtleStyle.Render(" ? help · j/k · m model · s start · space jump · i prompt · q quit")
	}

	if m.modelMode {
		// Model picker panel
		var pb strings.Builder
		id := m.activeID
		if len(m.agents) > 0 {
			id = m.agents[m.selected].ID
		}
		pb.WriteString(sectionStyle.Render(fmt.Sprintf(" MODEL · %s ", id)) + "\n\n")
		for i, name := range m.modelList {
			prefix := "  "
			line := name
			if i == m.modelIdx {
				prefix = activeBar.Render("▌") + " "
				line = selectedStyle.Render(name)
			}
			pb.WriteString(prefix + line + "\n")
		}
		pb.WriteString("\n" + subtleStyle.Render(" j/k move  enter set  esc cancel"))
		picker := rightStyle.Width(m.width - 4).Render(pb.String())
		sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#27272A")).Render(strings.Repeat("─", max(m.width, 1)))
		return lipgloss.JoinVertical(lipgloss.Left, header, sep, picker, status, helpView)
	}

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#27272A")).Render(strings.Repeat("─", max(m.width, 1)))
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		sep,
		body,
		inputBox,
		status,
		helpView,
	)
}

func (m Model) renderAgentList() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(" AGENTS ") + "\n")

	for i, a := range m.agents {
		selected := i == m.selected

		var statusDot string
		switch a.Status {
		case process.StatusRunning:
			statusDot = statusRun.Render("●")
		case process.StatusWaiting:
			statusDot = statusWait.Render("◉")
		case process.StatusError:
			statusDot = statusErr.Render("✗")
		case process.StatusMissing:
			statusDot = statusMiss.Render("○")
		case process.StatusStopped:
			statusDot = statusIdle.Render("○")
		default:
			statusDot = statusIdle.Render("○")
		}

		name := a.Name
		if len(name) > 16 {
			name = name[:14] + "…"
		}
		if selected {
			name = selectedStyle.Render(name)
		}

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
			}
		} else if a.Available && a.CmdPath != "" && a.Status == process.StatusIdle {
			st = a.ShortPath()
			if a.UsePTY {
				st += " · pty"
			}
		}
		if a.Model != "" {
			if st != "" {
				st += " · "
			}
			st += a.Model
		}
		if dir := m.manager.EffectiveCwd(a.ID); dir != "" {
			if d := m.cachedDiffShort(dir); d != "" && d != "clean" {
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

// Styles — dark violet cockpit palette
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#6D28D9")).
			Padding(0, 1)

	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E9D5FF")).
			Background(lipgloss.Color("#3B0764")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))

	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F5F3FF"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#C4B5FD")).
			Background(lipgloss.Color("#16121F")).
			Padding(0, 1)

	leftStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#3F3F46")).
			Padding(0, 1, 0, 0)

	rightStyle = lipgloss.NewStyle().
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A1A1AA")).
			Background(lipgloss.Color("#09090B"))

	statusIdle = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
	statusRun  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ADE80"))
	statusWait = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
	statusErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	statusMiss = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))

	fleetLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B")).Italic(true)
	promptLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD")).Bold(true)

	activeBar   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	selectedRow = lipgloss.NewStyle().Background(lipgloss.Color("#1A1229"))
)

