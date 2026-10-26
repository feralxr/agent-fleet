package ui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/feralxr/agent-fleet/internal/config"
	"github.com/feralxr/agent-fleet/internal/process"
)

const (
	tickInterval = 500 * time.Millisecond
)

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
	tmplMode     bool
	tmplIdx      int
	focusMode    bool
	noteMode     bool
	compareMode  bool
	pinArm       bool // waiting for 1-4 after '
	paletteMode  bool
	paletteQuery string
	paletteIdx   int
	paletteItems []paletteAction
	fanTargets   map[config.AgentID]bool
	histIdx      int // -1 = not browsing history
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
		fanTargets: make(map[config.AgentID]bool),
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

type modelsMsg struct {
	id     config.AgentID
	list   []string
	source string
}

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
