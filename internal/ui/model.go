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
	"github.com/charmbracelet/glamour"
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
	renderer    *glamour.TermRenderer
	width       int
	height      int
	showHelp    bool
	inputFocus  bool
	statusMsg   string
	lastTick    time.Time
	ready       bool
	lastContent string // avoid unnecessary re-renders
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

	// Soft dark theme that works well inside most terminals.
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	m := Model{
		manager:    mgr,
		agents:     agents,
		selected:   0,
		activeID:   agents[0].ID,
		viewport:   vp,
		input:      ta,
		help:       h,
		keys:       defaultKeyMap(),
		renderer:   r,
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

	// Fixed left sidebar, flexible output pane.
	leftWidth := 26
	if m.width < 90 {
		leftWidth = 22
	}
	if m.width < 70 {
		leftWidth = 18
	}

	// Header (1) + gap + body + gap + input (3) + status (1) + help (1)
	chrome := 9
	contentHeight := m.height - chrome
	if contentHeight < 6 {
		contentHeight = 6
	}

	rightWidth := m.width - leftWidth - 1
	if rightWidth < 20 {
		rightWidth = 20
	}

	m.viewport.Width = rightWidth
	m.viewport.Height = contentHeight
	m.input.SetWidth(m.width - 2)
	m.input.SetHeight(3)

	// Keep markdown wrapping in sync with the output pane.
	if m.renderer != nil && rightWidth > 12 {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(rightWidth-4),
		)
		if err == nil {
			m.renderer = r
			m.lastContent = ""
		}
	}
}

func (m *Model) refreshViewport() {
	p := m.manager.Get(m.activeID)
	if p == nil {
		m.setViewportContent("(no agent selected)")
		return
	}

	lines := p.SnapshotOutput()
	if len(lines) == 0 {
		msg := fmt.Sprintf("%s is **%s**", p.Name, p.Status)
		if p.Status == process.StatusMissing {
			msg += fmt.Sprintf("\n\nBinary not found: `%s`\n\nInstall it or set `FLEET_%s_CMD`", p.CmdPath, strings.ToUpper(string(p.ID)))
		} else {
			msg += "\n\nPress `s` to start."
		}
		m.setViewportContent(m.renderMarkdown(msg))
		return
	}

	// Build content: style system/fleet lines differently, keep agent output clean.
	var b strings.Builder
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

	// Try markdown rendering on the whole buffer for code blocks etc.
	// Fall back to styled plain text if glamour fails or is too slow on huge buffers.
	if len(lines) < 400 && m.renderer != nil {
		if rendered, err := m.renderer.Render(content); err == nil {
			content = rendered
		}
	}

	m.setViewportContent(content)
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

	// ── Header ──────────────────────────────────────────────
	title := titleStyle.Render(" Agent Fleet ")
	subtitle := subtleStyle.Render("  multi-agent switcher")
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle)

	// ── Body (sidebar + output) ─────────────────────────────
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
	right := m.viewport.View()

	leftPane := leftStyle.Width(leftWidth).Height(m.viewport.Height).Render(left)
	rightPane := rightStyle.Width(rightWidth).Height(m.viewport.Height).Render(right)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// ── Input ───────────────────────────────────────────────
	inputBorderColor := lipgloss.Color("#374151")
	if m.inputFocus {
		inputBorderColor = lipgloss.Color("#7C3AED")
	}
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(inputBorderColor).
		Padding(0, 1).
		Width(m.width - 2).
		Render(m.input.View())

	// ── Status bar ──────────────────────────────────────────
	cwd := m.manager.Cwd()
	if len(cwd) > 40 {
		cwd = "…" + cwd[len(cwd)-37:]
	}
	activeName := ""
	if p := m.manager.Get(m.activeID); p != nil {
		activeName = p.Name
	}
	statusLeft := fmt.Sprintf(" %s", m.statusMsg)
	statusRight := fmt.Sprintf("%s  ·  %s  ·  %s ", activeName, cwd, time.Now().Format("15:04:05"))
	gap := m.width - lipgloss.Width(statusLeft) - lipgloss.Width(statusRight)
	if gap < 1 {
		gap = 1
	}
	status := statusStyle.Width(m.width).Render(
		statusLeft + strings.Repeat(" ", gap) + statusRight,
	)

	// ── Help ────────────────────────────────────────────────
	var helpView string
	if m.showHelp {
		helpView = m.help.View(m.keys)
	} else {
		helpView = subtleStyle.Render(" ? help  ·  s start  ·  x stop  ·  r restart  ·  i input  ·  ctrl+l clear  ·  q quit")
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
	statusErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	statusMiss = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))

	fleetLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B")).Italic(true)
	promptLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD")).Bold(true)

	activeBar   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	selectedRow = lipgloss.NewStyle().Background(lipgloss.Color("#1E1B2E"))
)
