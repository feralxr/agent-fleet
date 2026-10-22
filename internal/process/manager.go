package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/feralxr/agent-fleet/internal/config"
)

// Status represents the current state of an agent process.
type Status string

const (
	StatusIdle    Status = "idle"
	StatusRunning Status = "running"
	StatusError   Status = "error"
	StatusStopped Status = "stopped"
)

// AgentProcess tracks a single running (or stopped) agent.
type AgentProcess struct {
	ID      config.AgentID
	Name    string
	Status  Status
	Cmd     *exec.Cmd
	Output  []string // recent output lines
	mu      sync.Mutex
	cancel  chan struct{}
	started time.Time
}

// Manager owns all agent processes.
type Manager struct {
	agents map[config.AgentID]*AgentProcess
	mu     sync.RWMutex
	cwd    string
}

// NewManager creates a process manager rooted at the given working directory.
func NewManager(cwd string) *Manager {
	m := &Manager{
		agents: make(map[config.AgentID]*AgentProcess),
		cwd:    cwd,
	}

	for _, a := range config.DefaultAgents() {
		m.agents[a.ID] = &AgentProcess{
			ID:     a.ID,
			Name:   a.Name,
			Status: StatusIdle,
			Output: make([]string, 0, 200),
		}
	}
	return m
}

// List returns a snapshot of all agents.
func (m *Manager) List() []*AgentProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*AgentProcess, 0, len(m.agents))
	for _, a := range config.DefaultAgents() {
		if p, ok := m.agents[a.ID]; ok {
			out = append(out, p)
		}
	}
	return out
}

// Get returns a single agent process.
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

	cfg := findConfig(id)
	if cfg == nil {
		return fmt.Errorf("no config for agent %s", id)
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = m.cwd
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		p.Status = StatusError
		p.append(fmt.Sprintf("[error] failed to start: %v", err))
		return err
	}

	p.Cmd = cmd
	p.Status = StatusRunning
	p.started = time.Now()
	p.cancel = make(chan struct{})
	p.append(fmt.Sprintf("[fleet] started %s (pid %d)", cfg.Name, cmd.Process.Pid))

	go m.stream(p, stdout)
	go m.stream(p, stderr)
	go m.wait(p)

	return nil
}

// Stop terminates a running agent.
func (m *Manager) Stop(id config.AgentID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("unknown agent: %s", id)
	}
	if p.Status != StatusRunning || p.Cmd == nil || p.Cmd.Process == nil {
		p.Status = StatusStopped
		return nil
	}

	close(p.cancel)
	_ = p.Cmd.Process.Kill()
	p.Status = StatusStopped
	p.append("[fleet] stopped")
	return nil
}

// Send writes a prompt to the agent's stdin if it is running.
// Note: many agents are interactive; this is a best-effort interface.
func (m *Manager) Send(id config.AgentID, text string) error {
	m.mu.RLock()
	p := m.agents[id]
	m.mu.RUnlock()

	if p == nil || p.Status != StatusRunning || p.Cmd == nil {
		return fmt.Errorf("agent %s is not running", id)
	}

	stdin, err := p.Cmd.StdinPipe()
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdin, text+"\n")
	return err
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

func (m *Manager) stream(p *AgentProcess, r io.Reader) {
	scanner := bufio.NewScanner(r)
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
	err := p.Cmd.Wait()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Status == StatusRunning {
		if err != nil {
			p.Status = StatusError
			p.append(fmt.Sprintf("[fleet] exited with error: %v", err))
		} else {
			p.Status = StatusStopped
			p.append("[fleet] exited cleanly")
		}
	}
}

func (p *AgentProcess) append(line string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Output = append(p.Output, line)
	if len(p.Output) > 500 {
		p.Output = p.Output[len(p.Output)-400:]
	}
}

// SnapshotOutput returns a copy of recent output lines.
func (p *AgentProcess) SnapshotOutput() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.Output))
	copy(out, p.Output)
	return out
}

func findConfig(id config.AgentID) *config.AgentConfig {
	for _, a := range config.DefaultAgents() {
		if a.ID == id {
			return &a
		}
	}
	return nil
}
