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
	Cwd      key.Binding
	Split    key.Binding
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
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Start, k.Stop, k.Restart, k.Cwd, k.Split, k.FocusIn, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Start, k.Stop, k.Restart},
		{k.FocusIn, k.FocusOut, k.Enter, k.Clear, k.Copy},
		{k.Cwd, k.Split, k.Help, k.Quit},
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
	split        bool // side-by-side output of two agents
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
		split:      false,
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
		// persist session
		m.session.LastCwd = m.manager.Cwd()
		m.session.LastAgentID = m.activeID
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
		m.input.Blur()
		m.input.Reset()
		m.input.Placeholder = "Type a prompt and press Enter to send to the active agent…"
		m.statusMsg = "cancelled"
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
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

	lines := p.SnapshotOutput()
	if len(lines) == 0 {
		msg := fmt.Sprintf("%s is **%s**", p.Name, p.Status)
		if p.Status == process.StatusMissing {
			msg += fmt.Sprintf("\n\nBinary not found: `%s`\n\nInstall it or set `FLEET_%s_CMD`", p.CmdPath, strings.ToUpper(string(p.ID)))
		} else {
			msg += "\n\nPress `s` to start."
		}
		content := m.renderMarkdown(msg)
		if content != *last {
			*last = content
			vp.SetContent(content)
			vp.GotoBottom()
		}
		return
	}

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
	if len(lines) < 400 && m.renderer != nil {
		if rendered, err := m.renderer.Render(content); err == nil {
			content = rendered
		}
	}
	if content != *last {
		*last = content
		vp.SetContent(content)
		vp.GotoBottom()
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
	mode := ""
	if m.split {
		mode = subtleStyle.Render("  split")
	}
	if m.pathMode {
		mode = subtleStyle.Render("  set cwd")
	}
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, mode)

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
	statusLeft := fmt.Sprintf(" %s", m.statusMsg)
	statusRight := fmt.Sprintf("%s  ·  %s  ·  %s ", activeName, cwd, time.Now().Format("15:04:05"))
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
		helpView = subtleStyle.Render(" ? help  ·  s start  ·  x stop  ·  r restart  ·  d cwd  ·  v split  ·  i input  ·  c copy  ·  q quit")
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
			if a.CmdPath != "" {
				st = "missing: " + a.ShortPath()
			}
		} else if a.Available && a.CmdPath != "" && a.Status == process.StatusIdle {
			st = a.ShortPath()
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
