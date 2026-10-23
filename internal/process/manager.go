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

	"github.com/feralxr/agent-fleet/internal/config"
)

// Status represents the current state of an agent process.
type Status string

const (
	StatusIdle      Status = "idle"
	StatusRunning   Status = "running"
	StatusWaiting   Status = "waiting" // running but likely needs user input
	StatusError     Status = "error"
	StatusStopped   Status = "stopped"
	StatusMissing   Status = "missing" // binary not found on PATH
)

const (
	maxOutputLines = 800
	trimToLines    = 600
)

// AgentProcess tracks a single agent instance.
type AgentProcess struct {
	ID          config.AgentID
	Name        string
	Status      Status
	Available   bool // true if the binary exists on PATH
	CmdPath     string
	WorktreePath string // optional isolated git worktree

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel chan struct{}
	done   chan struct{} // closed when the process has fully exited

	Output  []string
	mu      sync.Mutex
	started time.Time
	lastErr string
}

// Manager owns all agent processes.
type Manager struct {
	agents  map[config.AgentID]*AgentProcess
	configs []config.AgentConfig
	mu      sync.RWMutex
	cwd     string
}

// NewManager creates a process manager rooted at the given working directory.
// Pass a session to apply custom command overrides; nil is fine.
func NewManager(cwd string, session *config.Session) *Manager {
	cfgs := config.DefaultAgentsWithSession(session)
	m := &Manager{
		agents:  make(map[config.AgentID]*AgentProcess),
		configs: cfgs,
		cwd:     cwd,
	}

	for _, a := range cfgs {
		p := &AgentProcess{
			ID:     a.ID,
			Name:   a.Name,
			Status: StatusIdle,
			Output: make([]string, 0, 128),
		}
		if session != nil && session.Worktrees != nil {
			if wt, ok := session.Worktrees[string(a.ID)]; ok && wt != "" {
				if st, err := os.Stat(wt); err == nil && st.IsDir() {
					p.WorktreePath = wt
				}
			}
		}
		p.probeWith(a)
		m.agents[a.ID] = p
	}
	return m
}

// List returns agents in stable order.
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

// Get returns a single agent process (may be nil).
func (m *Manager) Get(id config.AgentID) *AgentProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[id]
}

// Start launches the given agent if it is not already running.
func (m *Manager) Start(id config.AgentID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("unknown agent: %s", id)
	}

	if p.Status == StatusRunning {
		return nil
	}

	// Re-probe in case the binary appeared after startup.
	p.probe(m)
	if !p.Available {
		p.Status = StatusMissing
		p.append(fmt.Sprintf("[fleet] binary not found: %s", p.CmdPath))
		return fmt.Errorf("%s binary not found on PATH (%s)", p.Name, p.CmdPath)
	}

	cfg := m.findConfig(id)
	if cfg == nil {
		return fmt.Errorf("no config for agent %s", id)
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	runDir := m.cwd
	if p.WorktreePath != "" {
		runDir = p.WorktreePath
	}
	cmd.Dir = runDir
	cmd.Env = os.Environ()

	// Create all pipes *before* Start so we keep a live stdin handle.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("stderr pipe: %w", err)
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
	p.Status = StatusRunning
	p.started = time.Now()
	p.cancel = make(chan struct{})
	p.done = make(chan struct{})
	p.lastErr = ""
	cmdLine := cfg.Command
	if len(cfg.Args) > 0 {
		cmdLine += " " + strings.Join(cfg.Args, " ")
	}
	p.append(fmt.Sprintf("[fleet] started %s — %s (pid %d) in %s", p.Name, cmdLine, cmd.Process.Pid, runDir))

	go m.stream(p, stdout, "out")
	go m.stream(p, stderr, "err")
	go m.wait(p)

	return nil
}

// Stop terminates a running agent cleanly.
func (m *Manager) Stop(id config.AgentID) error {
	m.mu.Lock()
	p, ok := m.agents[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown agent: %s", id)
	}

	if p.Status != StatusRunning || p.cmd == nil || p.cmd.Process == nil {
		if p.Status == StatusRunning {
			p.Status = StatusStopped
		}
		m.mu.Unlock()
		return nil
	}

	// Signal the streaming goroutines to stop.
	select {
	case <-p.cancel:
		// already closed
	default:
		close(p.cancel)
	}

	// Close stdin so the agent sees EOF (helps graceful shutdown for some tools).
	if p.stdin != nil {
		_ = p.stdin.Close()
		p.stdin = nil
	}

	// Ask the process to exit, then force-kill if needed.
	_ = p.cmd.Process.Signal(os.Interrupt)

	cmd := p.cmd
	done := p.done
	m.mu.Unlock()

	// Wait a short time for graceful exit, then kill.
	select {
	case <-done:
		// exited cleanly
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

// Restart stops (if running) then starts the agent.
func (m *Manager) Restart(id config.AgentID) error {
	_ = m.Stop(id)
	// tiny pause so the OS releases the process table entry
	time.Sleep(100 * time.Millisecond)
	return m.Start(id)
}

// Send writes a prompt to the agent's stdin.
// The newline is always appended.
func (m *Manager) Send(id config.AgentID, text string) error {
	m.mu.RLock()
	p := m.agents[id]
	m.mu.RUnlock()

	if p == nil {
		return fmt.Errorf("unknown agent: %s", id)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != StatusRunning || p.stdin == nil {
		return fmt.Errorf("%s is not running", p.Name)
	}

	_, err := io.WriteString(p.stdin, text+"\n")
	if err != nil {
		p.lastErr = err.Error()
		return fmt.Errorf("write to %s: %w", p.Name, err)
	}

	// Echo the prompt into the output buffer so the user sees what was sent.
	p.Output = append(p.Output, fmt.Sprintf("› %s", text))
	if len(p.Output) > maxOutputLines {
		p.Output = p.Output[len(p.Output)-trimToLines:]
	}
	if p.Status == StatusWaiting {
		p.Status = StatusRunning
	}
	return nil
}

// ClearOutput wipes the output buffer for an agent.
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

// NextNeedingAttention returns the index in List() order of the next agent
// that needs attention after afterID (wrapping). Returns -1 if none.
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

// SetWorktree assigns an isolated worktree path to an agent (empty clears it).
func (m *Manager) SetWorktree(id config.AgentID, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.agents[id]; ok {
		p.WorktreePath = path
	}
}

// Worktree returns the worktree path for an agent (may be empty).
func (m *Manager) Worktree(id config.AgentID) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.agents[id]; ok {
		return p.WorktreePath
	}
	return ""
}

// EffectiveCwd is the directory the agent will run in (worktree or shared cwd).
func (m *Manager) EffectiveCwd(id config.AgentID) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.agents[id]; ok && p.WorktreePath != "" {
		return p.WorktreePath
	}
	return m.cwd
}

// SetCwd updates the working directory used for new processes.
func (m *Manager) SetCwd(cwd string) {
	m.mu.Lock()
	m.cwd = cwd
	m.mu.Unlock()
}

// Cwd returns the current working directory.
func (m *Manager) Cwd() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cwd
}

// LastError returns the most recent error string for an agent (may be empty).
func (p *AgentProcess) LastError() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastErr
}

// SnapshotOutput returns a copy of recent output lines.
func (p *AgentProcess) SnapshotOutput() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.Output))
	copy(out, p.Output)
	return out
}

// probeWith checks whether the configured binary exists.
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

// probe re-checks using the manager's config list.
func (p *AgentProcess) probe(m *Manager) {
	cfg := m.findConfig(p.ID)
	if cfg == nil {
		p.Available = false
		p.CmdPath = ""
		p.Status = StatusMissing
		return
	}
	p.probeWith(*cfg)
}

func (m *Manager) stream(p *AgentProcess, r io.Reader, _ string) {
	scanner := bufio.NewScanner(r)
	// Increase buffer for long lines (agent output can be verbose).
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		select {
		case <-p.cancel:
			return
		default:
			p.append(scanner.Text())
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

	if p.Status == StatusRunning {
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

	// Signal that the process has fully exited.
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
	if len(p.Output) > maxOutputLines {
		p.Output = p.Output[len(p.Output)-trimToLines:]
	}
	// Soft heuristic: if the process is running and the latest lines look like a prompt, mark waiting.
	if p.Status == StatusRunning || p.Status == StatusWaiting {
		if looksLikePrompt(line) {
			p.Status = StatusWaiting
		} else if p.Status == StatusWaiting && !looksLikePrompt(line) {
			// New non-prompt output → back to running
			p.Status = StatusRunning
		}
	}
}

// looksLikePrompt detects common "agent is waiting for you" patterns.
func looksLikePrompt(line string) bool {
	s := strings.TrimSpace(strings.ToLower(line))
	if s == "" {
		return false
	}
	patterns := []string{
		"y/n",
		"(y/n)",
		"[y/n]",
		"yes/no",
		"allow?",
		"approve",
		"permission",
		"press enter",
		"continue?",
		"confirm",
		"waiting for",
		"your input",
		"type a message",
		"> ",
		"❯ ",
		"? ",
		"do you want",
		"shall i",
		"may i",
		"ok to",
		"proceed?",
	}
	for _, ptn := range patterns {
		if strings.Contains(s, ptn) {
			return true
		}
	}
	// Ends with ? and short → likely a question
	if strings.HasSuffix(s, "?") && len(s) < 120 {
		return true
	}
	return false
}

// NeedsAttention reports whether the agent is blocked on the user.
func (p *AgentProcess) NeedsAttention() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Status == StatusWaiting || p.Status == StatusError
}

// MarkRunning forces status back to running (e.g. after user sent input).
func (p *AgentProcess) MarkRunning() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Status == StatusWaiting {
		p.Status = StatusRunning
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

// Helper so UI can show a short path.
func (p *AgentProcess) ShortPath() string {
	if p.CmdPath == "" {
		return ""
	}
	return filepath.Base(p.CmdPath)
}
