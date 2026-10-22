package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/feralxr/agent-fleet/internal/config"
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
	Restart  key.Binding
	Clear    key.Binding
	FocusIn  key.Binding
	FocusOut key.Binding
	Copy     key.Binding
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
			key.WithHelp("enter", "send prompt"),
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
			key.WithHelp("esc", "unfocus input"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy last output"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Start, k.Stop, k.Restart, k.FocusIn, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Start, k.Stop, k.Restart},
		{k.FocusIn, k.FocusOut, k.Enter, k.Clear},
		{k.Help, k.Quit},
	}
}

// Model is the main Bubble Tea model.
type Model struct {
	manager     *process.Manager
	agents      []*process.AgentProcess
	selected    int
	activeID    config.AgentID
	viewport    viewport.Model
	input       textarea.Model
	help        help.Model
	keys        keyMap
	width       int
	height      int
	showHelp    bool
	inputFocus  bool
	statusMsg   string
	lastTick    time.Time
	ready       bool
}

// NewModel creates the initial UI model.
func NewModel() Model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	mgr := process.NewManager(cwd)
	agents := mgr.List()

	ta := textarea.New()
	ta.Placeholder = "Type a prompt and press Enter to send to the active agent…"
	ta.CharLimit = 4000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Focus()

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to Agent Fleet\n\nSelect an agent on the left and press 's' to start it.")

	h := help.New()

	m := Model{
		manager:    mgr,
		agents:     agents,
		selected:   0,
		activeID:   agents[0].ID,
		viewport:   vp,
		input:      ta,
		help:       h,
		keys:       defaultKeyMap(),
		inputFocus: false,
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

	case key.Matches(msg, m.keys.FocusIn):
		m.inputFocus = true
		m.input.Focus()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Copy):
		m.statusMsg = "copy not implemented yet (coming soon)"
		return m, nil
	}

	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.FocusOut):
		m.inputFocus = false
		m.input.Blur()
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		err := m.manager.Send(m.activeID, text)
		if err != nil {
			m.statusMsg = fmt.Sprintf("send failed: %v", err)
		} else {
			m.statusMsg = "prompt sent"
			m.input.Reset()
		}
		m.refreshViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// left pane ~ 22 cols, rest for output, bottom for input + status
	leftWidth := 24
	if m.width < 80 {
		leftWidth = 18
	}
	rightWidth := m.width - leftWidth - 2
	if rightWidth < 20 {
		rightWidth = 20
	}

	contentHeight := m.height - 8 // header + input + status + help
	if contentHeight < 5 {
		contentHeight = 5
	}

	m.viewport.Width = rightWidth
	m.viewport.Height = contentHeight
	m.input.SetWidth(m.width - 4)
}

func (m *Model) refreshViewport() {
	p := m.manager.Get(m.activeID)
	if p == nil {
		m.viewport.SetContent("(no agent selected)")
		return
	}
	lines := p.SnapshotOutput()
	if len(lines) == 0 {
		msg := fmt.Sprintf("%s is %s", p.Name, p.Status)
		if p.Status == process.StatusMissing {
			msg += fmt.Sprintf("\n\nBinary not found: %s\nInstall it or set FLEET_%s_CMD", p.CmdPath, strings.ToUpper(string(p.ID)))
		} else {
			msg += "\n\nPress 's' to start."
		}
		m.viewport.SetContent(msg)
		return
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	if !m.ready {
		return "Loading…"
	}

	header := titleStyle.Render(" Agent Fleet ") + "  " +
		subtleStyle.Render("Cursor · OpenCode · Claude Code · Grok")

	left := m.renderAgentList()
	right := m.viewport.View()

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Width(24).Render(left),
		rightStyle.Width(m.width-26).Render(right),
	)

	inputView := m.input.View()
	if !m.inputFocus {
		inputView = subtleStyle.Render(inputView)
	}

	status := statusStyle.Render(fmt.Sprintf(" %s  │  cwd: %s  │  %s ",
		m.statusMsg, m.manager.Cwd(), time.Now().Format("15:04:05")))

	helpView := ""
	if m.showHelp {
		helpView = "\n" + m.help.View(m.keys)
	} else {
		helpView = "\n" + subtleStyle.Render(" ? help │ s start │ x stop │ r restart │ i input │ ctrl+l clear │ q quit ")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		inputStyle.Render(inputView),
		status,
		helpView,
	)
}

func (m Model) renderAgentList() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render(" AGENTS ") + "\n\n")

	for i, a := range m.agents {
		cursor := "  "
		name := a.Name
		if i == m.selected {
			cursor = "▸ "
			name = selectedStyle.Render(a.Name)
		}

		var status string
		switch a.Status {
		case process.StatusRunning:
			status = statusRun.Render("● running")
		case process.StatusError:
			status = statusErr.Render("● error")
		case process.StatusMissing:
			status = statusMiss.Render("● missing")
		case process.StatusStopped:
			status = statusIdle.Render("● stopped")
		default:
			status = statusIdle.Render("● idle")
		}

		line := fmt.Sprintf("%s%s\n   %s\n", cursor, name, status)
		b.WriteString(line)
	}

	b.WriteString("\n" + subtleStyle.Render("↑↓ select · s start · r restart"))
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

	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#C4B5FD"))

	leftStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(0, 1)

	rightStyle = lipgloss.NewStyle().
			Padding(0, 1)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Background(lipgloss.Color("#111827"))

	statusIdle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	statusRun  = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399"))
	statusErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	statusMiss = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
)
