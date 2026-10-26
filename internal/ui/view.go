package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/feralxr/agent-fleet/internal/config"
	"github.com/feralxr/agent-fleet/internal/gitutil"
	"github.com/feralxr/agent-fleet/internal/process"
)

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
	if m.focusMode {
		leftWidth = 0
	}

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
		secIdx := m.selected + 1
		if !m.compareMode {
			secIdx = m.selected - 1
			if secIdx < 0 {
				secIdx = m.selected + 1
			}
		}
		if secIdx >= len(m.agents) {
			secIdx = 0
		}
		if secIdx == m.selected && len(m.agents) > 1 {
			secIdx = (m.selected + 1) % len(m.agents)
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
	if m.tmplMode {
		badges = append(badges, "tmpl")
	}
	if m.focusMode {
		badges = append(badges, "focus")
	}
	if m.compareMode {
		badges = append(badges, "compare")
	}
	if m.noteMode {
		badges = append(badges, "note")
	}
	if m.pinArm {
		badges = append(badges, "pin")
	}
	if m.paletteMode {
		badges = append(badges, "cmd")
	}
	if len(m.fanTargets) > 0 {
		badges = append(badges, fmt.Sprintf("fan:%d", len(m.fanTargets)))
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
	if m.focusMode {
		leftWidth = 0
		rightWidth = m.width
		leftPane = ""
	}

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
	case m.noteMode:
		inputBorderColor = lipgloss.Color("#FBBF24")
		inputLabel = statusMiss.Render(" note ")
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
		helpView = subtleStyle.Render(" ? · T tmpl · n note · o compare · z focus · m model · p resend")
	}

	if m.tmplMode {
		var pb strings.Builder
		pb.WriteString(sectionStyle.Render(" TEMPLATES ") + "\n\n")
		for i, tmpl := range promptTemplates {
			prefix := "  "
			line := fmt.Sprintf("/%-10s %s", tmpl.ID, tmpl.Label)
			if i == m.tmplIdx {
				prefix = activeBar.Render("▌") + " "
				line = selectedStyle.Render(line)
			}
			pb.WriteString(prefix + line + "\n")
		}
		pb.WriteString("\n" + subtleStyle.Render(" type /review /tests /explain in prompt too"))
		picker := rightStyle.Width(m.width - 4).Render(pb.String())
		sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#27272A")).Render(strings.Repeat("─", max(m.width, 1)))
		return lipgloss.JoinVertical(lipgloss.Left, header, sep, picker, status, helpView)
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
		if m.session.Notes != nil {
			if note := m.session.Notes[string(a.ID)]; note != "" {
				if len(note) > 18 {
					note = note[:16] + "…"
				}
				if st != "" {
					st += " · "
				}
				st += "note:" + note
			}
		}
		if a.Status == process.StatusRunning || a.Status == process.StatusWaiting {
			act := a.ActivityLabel()
			if act != "" {
				if st != "" {
					st += " · "
				}
				st += act
			}
		}
		if dir := m.manager.EffectiveCwd(a.ID); dir != "" {
			if d := m.cachedDiffShort(dir); d != "" && d != "clean" {
				if st != "" {
					st += " · "
				}
				st += d
			}
		}
		spin := ""
		if a.Status == process.StatusRunning && a.IdleFor() < 2*time.Second {
			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			spin = frames[int(time.Now().UnixMilli()/120)%len(frames)] + " "
		}
		sec := subtleStyle.Render(fmt.Sprintf("    %s%s", spin, st))
		b.WriteString(sec + "\n")
	}

	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(" ● run  ◉ wait  ✗ err") + "\n")
	b.WriteString(subtleStyle.Render(" p resend  e export") + "\n")
	return b.String()
}
