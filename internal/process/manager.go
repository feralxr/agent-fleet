package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/feralxr/agent-fleet/internal/config"
)

type Status string

const (
	StatusIdle    Status = "idle"
	StatusRunning Status = "running"
	StatusWaiting Status = "waiting"
	StatusError   Status = "error"
	StatusStopped Status = "stopped"
	StatusMissing Status = "missing"
)

const (
	maxOutputLines = 800
	trimToLines    = 600
)

type AgentProcess struct {
	ID           config.AgentID
	Name         string
	Status       Status
	Available    bool
	CmdPath      string
	WorktreePath string
	UsePTY       bool
	Model        string // selected model (passed as --model on start)

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	ptmx   *os.File
	cancel chan struct{}
	done   chan struct{}

	Output     []string
	mu         sync.Mutex
	started    time.Time
	lastActive time.Time
	lastErr    string
}

type Manager struct {
	agents  map[config.AgentID]*AgentProcess
	configs []config.AgentConfig
	mu      sync.RWMutex
	cwd     string
}

func NewManager(cwd string, session *config.Session) *Manager {
	cfgs := config.DefaultAgentsWithSession(session)
	m := &Manager{
		agents:  make(map[config.AgentID]*AgentProcess),
		configs: cfgs,
		cwd:     cwd,
	}
	for _, a := range cfgs {
		p := &AgentProcess{
			ID: a.ID, Name: a.Name, Status: StatusIdle, UsePTY: a.UsePTY,
			Output: make([]string, 0, 128),
			Model:  a.Model,
		}
		if session != nil {
			if session.Worktrees != nil {
				if wt, ok := session.Worktrees[string(a.ID)]; ok && wt != "" {
					if st, err := os.Stat(wt); err == nil && st.IsDir() {
						p.WorktreePath = wt
					}
				}
			}
			if session.Models != nil {
				if mod, ok := session.Models[string(a.ID)]; ok {
					p.Model = mod
				}
			}
		}
		p.probeWith(a)
		m.agents[a.ID] = p
	}
	return m
}

func (m *Manager) List() []*AgentProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*AgentProcess, 0, len(m.agents))
	for _, a := range m.configs {
		if p, ok := m.agents[a.ID]; ok {
			out = append(out, p)
		}
	}
	return out
}

func (m *Manager) Get(id config.AgentID) *AgentProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[id]
}

// ReloadConfigs reloads agent definitions from fleet.toml + session.
// Running agents keep their process; metadata (name, pty flag, availability) updates.
// New agent IDs appear; removed IDs stay until restart if still running.
func (m *Manager) ReloadConfigs(session *config.Session) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfgs := config.DefaultAgentsWithSession(session)
	m.configs = cfgs
	added := 0
	for _, a := range cfgs {
		if existing, ok := m.agents[a.ID]; ok {
			existing.Name = a.Name
			existing.UsePTY = a.UsePTY
			existing.probeWith(a)
			continue
		}
		p := &AgentProcess{
			ID: a.ID, Name: a.Name, Status: StatusIdle, UsePTY: a.UsePTY,
			Output: make([]string, 0, 128),
		}
		p.probeWith(a)
		m.agents[a.ID] = p
		added++
	}
	return added
}

// AttentionCount returns how many agents need attention.
func (m *Manager) AttentionCount() int {
	n := 0
	for _, p := range m.List() {
		if p.NeedsAttention() {
			n++
		}
	}
	return n
}

// RunningCount returns how many agents are running or waiting.
func (m *Manager) RunningCount() int {
	n := 0
	for _, p := range m.List() {
		st := p.Status
		if st == StatusRunning || st == StatusWaiting {
			n++
		}
	}
	return n
}

func (m *Manager) Start(id config.AgentID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("unknown agent: %s", id)
	}
	if p.Status == StatusRunning || p.Status == StatusWaiting {
		return nil
	}
	cfg := m.findConfig(id)
	if cfg == nil {
		return fmt.Errorf("no config for agent %s", id)
	}
	p.probeWith(*cfg)
	if !p.Available {
		p.Status = StatusMissing
		p.append(fmt.Sprintf("[fleet] binary not found: %s", p.CmdPath))
		return fmt.Errorf("%s binary not found on PATH (%s)", p.Name, p.CmdPath)
	}

	runDir := m.cwd
	if p.WorktreePath != "" {
		runDir = p.WorktreePath
	}
	args := append([]string{}, cfg.Args...)
	if model := strings.TrimSpace(p.Model); model != "" {
		args = append(args, config.ModelFlag(id), model)
	}
	cmd := exec.Command(cfg.Command, args...)
	cmd.Dir = runDir
	cmd.Env = os.Environ()

	cmdLine := cfg.Command
	if len(args) > 0 {
		cmdLine += " " + strings.Join(args, " ")
	}

	if cfg.UsePTY {
		ptmx, err := pty.Start(cmd)
		if err != nil {
			p.Status = StatusError
			p.lastErr = err.Error()
			p.append(fmt.Sprintf("[fleet] failed to start pty: %v", err))
			return err
		}
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 32, Cols: 120})
		p.cmd = cmd
		p.ptmx = ptmx
		p.stdin = nil
		p.Status = StatusRunning
		p.started = time.Now()
		p.lastActive = p.started
		p.cancel = make(chan struct{})
		p.done = make(chan struct{})
		p.lastErr = ""
		p.append(fmt.Sprintf("[fleet] started %s — %s (pid %d, pty) in %s", p.Name, cmdLine, cmd.Process.Pid, runDir))
		go m.stream(p, ptmx)
		go m.wait(p)
		return nil
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		p.Status = StatusError
		p.lastErr = err.Error()
		p.append(fmt.Sprintf("[fleet] failed to start: %v", err))
		return err
	}
	p.cmd = cmd
	p.stdin = stdin
	p.ptmx = nil
	p.Status = StatusRunning
	p.started = time.Now()
	p.lastActive = p.started
	p.cancel = make(chan struct{})
	p.done = make(chan struct{})
	p.lastErr = ""
	p.append(fmt.Sprintf("[fleet] started %s — %s (pid %d, pipes) in %s", p.Name, cmdLine, cmd.Process.Pid, runDir))
	go m.stream(p, stdout)
	go m.stream(p, stderr)
	go m.wait(p)
	return nil
}

func (m *Manager) Stop(id config.AgentID) error {
	m.mu.Lock()
	p, ok := m.agents[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown agent: %s", id)
	}
	if p.Status != StatusRunning && p.Status != StatusWaiting {
		m.mu.Unlock()
		return nil
	}
	if p.cmd == nil || p.cmd.Process == nil {
		p.Status = StatusStopped
		m.mu.Unlock()
		return nil
	}
	select {
	case <-p.cancel:
	default:
		close(p.cancel)
	}
	if p.stdin != nil {
		_ = p.stdin.Close()
		p.stdin = nil
	}
	if p.ptmx != nil {
		_ = p.ptmx.Close()
		p.ptmx = nil
	}
	_ = p.cmd.Process.Signal(os.Interrupt)
	cmd := p.cmd
	done := p.done
	m.mu.Unlock()

	select {
	case <-done:
	case <-time.After(1500 * time.Millisecond):
		_ = cmd.Process.Kill()
		<-done
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p.Status = StatusStopped
	p.append("[fleet] stopped")
	return nil
}

func (m *Manager) Restart(id config.AgentID) error {
	_ = m.Stop(id)
	time.Sleep(100 * time.Millisecond)
	return m.Start(id)
}

func (m *Manager) Send(id config.AgentID, text string) error {
	m.mu.RLock()
	p := m.agents[id]
	m.mu.RUnlock()
	if p == nil {
		return fmt.Errorf("unknown agent: %s", id)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Status != StatusRunning && p.Status != StatusWaiting {
		return fmt.Errorf("%s is not running", p.Name)
	}
	payload := text + "\n"
	var err error
	switch {
	case p.ptmx != nil:
		_, err = p.ptmx.Write([]byte(payload))
	case p.stdin != nil:
		_, err = io.WriteString(p.stdin, payload)
	default:
		return fmt.Errorf("%s has no writable input", p.Name)
	}
	if err != nil {
		p.lastErr = err.Error()
		return fmt.Errorf("write to %s: %w", p.Name, err)
	}
	p.Output = append(p.Output, fmt.Sprintf("› %s", text))
	if len(p.Output) > maxOutputLines {
		p.Output = p.Output[len(p.Output)-trimToLines:]
	}
	if p.Status == StatusWaiting {
		p.Status = StatusRunning
	}
	return nil
}

func (m *Manager) SetPTYSize(id config.AgentID, rows, cols uint16) {
	m.mu.RLock()
	p := m.agents[id]
	m.mu.RUnlock()
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ptmx != nil && rows > 0 && cols > 0 {
		_ = pty.Setsize(p.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
	}
}

func (m *Manager) ClearOutput(id config.AgentID) {
	m.mu.RLock()
	p := m.agents[id]
	m.mu.RUnlock()
	if p == nil {
		return
	}
	p.mu.Lock()
	p.Output = p.Output[:0]
	p.mu.Unlock()
}

func (m *Manager) NextNeedingAttention(afterID config.AgentID) config.AgentID {
	list := m.List()
	if len(list) == 0 {
		return ""
	}
	start := 0
	for i, a := range list {
		if a.ID == afterID {
			start = i + 1
			break
		}
	}
	for i := 0; i < len(list); i++ {
		idx := (start + i) % len(list)
		if list[idx].NeedsAttention() {
			return list[idx].ID
		}
	}
	return ""
}

func (m *Manager) SetWorktree(id config.AgentID, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.agents[id]; ok {
		p.WorktreePath = path
	}
}

func (m *Manager) Worktree(id config.AgentID) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.agents[id]; ok {
		return p.WorktreePath
	}
	return ""
}

func (m *Manager) EffectiveCwd(id config.AgentID) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.agents[id]; ok && p.WorktreePath != "" {
		return p.WorktreePath
	}
	return m.cwd
}

func (m *Manager) SetCwd(cwd string) {
	m.mu.Lock()
	m.cwd = cwd
	m.mu.Unlock()
}

func (m *Manager) Cwd() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cwd
}

func (p *AgentProcess) LastError() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastErr
}

func (p *AgentProcess) SnapshotOutput() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.Output))
	copy(out, p.Output)
	return out
}

// OutputFingerprint is a cheap change detector (no full buffer copy).
func (p *AgentProcess) OutputFingerprint() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.Output)
	if n == 0 {
		return string(p.Status)
	}
	last := p.Output[n-1]
	if len(last) > 64 {
		last = last[len(last)-64:]
	}
	return fmt.Sprintf("%s|%d|%s", p.Status, n, last)
}

func (p *AgentProcess) NeedsAttention() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Status == StatusWaiting || p.Status == StatusError
}

func (p *AgentProcess) ShortPath() string {
	if p.CmdPath == "" {
		return ""
	}
	return filepath.Base(p.CmdPath)
}

func (p *AgentProcess) probeWith(cfg config.AgentConfig) {
	path, err := exec.LookPath(cfg.Command)
	if err != nil {
		p.Available = false
		p.CmdPath = cfg.Command
		if p.Status == StatusIdle || p.Status == StatusStopped {
			p.Status = StatusMissing
		}
		return
	}
	p.Available = true
	p.CmdPath = path
	if p.Status == StatusMissing {
		p.Status = StatusIdle
	}
}

func (m *Manager) findConfig(id config.AgentID) *config.AgentConfig {
	for _, a := range m.configs {
		if a.ID == id {
			cp := a
			return &cp
		}
	}
	return nil
}

func (m *Manager) stream(p *AgentProcess, r io.Reader) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		select {
		case <-p.cancel:
			return
		default:
			line := stripANSI(scanner.Text())
			if line == "" {
				continue
			}
			p.append(line)
		}
	}
}

func (m *Manager) wait(p *AgentProcess) {
	err := p.cmd.Wait()
	p.mu.Lock()
	if p.stdin != nil {
		_ = p.stdin.Close()
		p.stdin = nil
	}
	if p.ptmx != nil {
		_ = p.ptmx.Close()
		p.ptmx = nil
	}
	if p.Status == StatusRunning || p.Status == StatusWaiting {
		if err != nil {
			p.Status = StatusError
			p.lastErr = err.Error()
			p.Output = append(p.Output, fmt.Sprintf("[fleet] exited with error: %v", err))
		} else {
			p.Status = StatusStopped
			p.Output = append(p.Output, "[fleet] exited cleanly")
		}
		if len(p.Output) > maxOutputLines {
			p.Output = p.Output[len(p.Output)-trimToLines:]
		}
	}
	p.mu.Unlock()
	select {
	case <-p.done:
	default:
		close(p.done)
	}
}

func (p *AgentProcess) append(line string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Output = append(p.Output, line)
	p.lastActive = time.Now()
	if len(p.Output) > maxOutputLines {
		p.Output = p.Output[len(p.Output)-trimToLines:]
	}
	if p.Status == StatusRunning || p.Status == StatusWaiting {
		if looksLikePrompt(line) {
			p.Status = StatusWaiting
		} else if p.Status == StatusWaiting {
			p.Status = StatusRunning
		}
	}
}

// Uptime returns how long the process has been running, or 0 if not.
func (p *AgentProcess) Uptime() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Status != StatusRunning && p.Status != StatusWaiting {
		return 0
	}
	if p.started.IsZero() {
		return 0
	}
	return time.Since(p.started)
}

// LastActive returns when output was last received.
func (p *AgentProcess) LastActive() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastActive
}

// FormatUptime is a human short uptime like 3m12s.
func FormatUptime(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%dm%02ds", s/60, s%60)
	}
	return fmt.Sprintf("%dh%02dm", s/3600, (s%3600)/60)
}

// IdleFor how long since last output while running/waiting.
func (p *AgentProcess) IdleFor() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Status != StatusRunning && p.Status != StatusWaiting {
		return 0
	}
	if p.lastActive.IsZero() {
		return 0
	}
	return time.Since(p.lastActive)
}

// ActivityLabel short UX string: active / thinking / idle.
func (p *AgentProcess) ActivityLabel() string {
	switch p.Status {
	case StatusWaiting:
		return "needs you"
	case StatusRunning:
		idle := p.IdleFor()
		if idle < 2*time.Second {
			return "streaming"
		}
		if idle < 15*time.Second {
			return "working"
		}
		return fmt.Sprintf("quiet %s", FormatUptime(idle))
	default:
		return string(p.Status)
	}
}

// SetModel sets the model for an agent (applied on next Start).
func (m *Manager) SetModel(id config.AgentID, model string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.agents[id]; ok {
		p.Model = strings.TrimSpace(model)
	}
}

// GetModel returns the selected model for an agent.
func (m *Manager) GetModel(id config.AgentID) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.agents[id]; ok {
		return p.Model
	}
	return ""
}

// ModelsFor returns the catalog for an agent: live CLI list + config + defaults.
func (m *Manager) ModelsFor(id config.AgentID) []string {
	cfg := m.findConfig(id)
	var live []string
	if cfg != nil {
		live = DiscoverModels(*cfg)
	}
	return MergeModels(id, cfg, live)
}

// ApplyModel sets the model and, if the agent is running, sends /model into the session.
func (m *Manager) ApplyModel(id config.AgentID, model string) error {
	model = strings.TrimSpace(model)
	m.SetModel(id, model)
	m.mu.RLock()
	p := m.agents[id]
	running := p != nil && (p.Status == StatusRunning || p.Status == StatusWaiting)
	m.mu.RUnlock()
	if !running || model == "" {
		return nil
	}
	// Most agent TUIs accept /model <id> as an in-session switch.
	return m.Send(id, "/model "+model)
}

// StopAll stops every running agent.
func (m *Manager) StopAll() int {
	n := 0
	for _, p := range m.List() {
		if p.Status == StatusRunning || p.Status == StatusWaiting {
			if err := m.Stop(p.ID); err == nil {
				n++
			}
		}
	}
	return n
}

func looksLikePrompt(line string) bool {
	s := strings.TrimSpace(strings.ToLower(line))
	if s == "" {
		return false
	}
	patterns := []string{
		"y/n", "(y/n)", "[y/n]", "yes/no", "allow?", "approve", "permission",
		"press enter", "continue?", "confirm", "waiting for", "your input",
		"do you want", "shall i", "may i", "ok to", "proceed?",
	}
	for _, ptn := range patterns {
		if strings.Contains(s, ptn) {
			return true
		}
	}
	return strings.HasSuffix(s, "?") && len(s) < 120
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) {
			if s[i+1] == '[' {
				i += 2
				for i < len(s) && !((s[i] >= '@' && s[i] <= '_') || (s[i] >= 'a' && s[i] <= 'z')) {
					i++
				}
				if i < len(s) {
					i++
				}
				continue
			}
			if s[i+1] == ']' {
				i += 2
				for i < len(s) && s[i] != 0x07 && s[i] != 0x1b {
					i++
				}
				if i < len(s) && s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i += 2
				} else if i < len(s) {
					i++
				}
				continue
			}
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
