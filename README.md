# Agent Fleet

Lightweight terminal UI for running and switching between multiple coding agents side-by-side.

Supports **Claude Code**, **Cursor**, **OpenCode**, and **Grok**.

## Features

- Launch / stop / restart any agent
- **Git worktrees per agent** — isolated checkouts so agents don't clash
- **Fan-out** — send one prompt to every running agent
- **Attention jump** — space hops to agents waiting for input
- **Diff review** — `g` shows `git diff --stat` per agent
- Split view (watch two agents at once)
- Change working directory from the TUI
- Shared prompt input + live markdown output
- Session persistence (cwd, active agent, worktrees)
- Clipboard copy
- Binary detection

## Install

```bash
go install github.com/feralxr/agent-fleet/cmd/fleet@latest
```

Or from source:

```bash
git clone https://github.com/feralxr/agent-fleet.git
cd agent-fleet
go build -o fleet ./cmd/fleet
```

## Usage

```bash
cd /path/to/your/repo
fleet
```

### Keybindings

| Key        | Action                                      |
|------------|---------------------------------------------|
| `↑` / `k`  | Previous agent                              |
| `↓` / `j`  | Next agent                                  |
| `s`        | Start selected agent                        |
| `x`        | Stop selected agent                         |
| `r`        | Restart selected agent                      |
| `space`    | Jump to next agent that needs attention     |
| `g`        | Toggle git diff --stat for selected agent   |
| `w`        | Create / attach git worktree for agent      |
| `f`        | Fan-out mode (next prompt → all running)    |
| `d`        | Change working directory                    |
| `v`        | Toggle split view                           |
| `i`        | Focus prompt input                          |
| `Esc`      | Cancel / unfocus                            |
| `Enter`    | Send prompt / confirm path / fan-out        |
| `c`        | Copy output to clipboard                    |
| `ctrl+l`   | Clear output                                |
| `?`        | Toggle help                                 |
| `q`        | Quit (saves session)                        |

### Worktrees

Press `w` on an agent while inside a git repo. Fleet creates:

```
<repo>/.fleet/worktrees/<agent-id>
```

on branch `fleet/<agent-id>`. That agent then runs with its cwd set to the worktree so parallel edits don't collide.

Add `.fleet/` to your repo's `.gitignore` if you want to keep worktrees local-only.

### Diff review

Press **`g`** to toggle a `git diff --stat` view for the selected agent's effective directory (worktree or shared cwd). Sidebar shows a one-line dirty summary when the tree is not clean. Press **`g`** again to return to live output.

### Attention

Agents show `◐` (cyan) when recent output looks like a question or approval prompt (`y/n`, "allow?", etc.). Press **space** to jump to the next agent that needs you. Sending a prompt clears the waiting state.

### Fan-out

1. Start two or more agents (`s`)
2. Press `f`
3. Type a prompt and press Enter — every running agent receives it

## Configuration

| Agent       | Default command | Override env var     |
|-------------|-----------------|----------------------|
| Claude Code | `claude`        | `FLEET_CLAUDE_CMD`   |
| Cursor      | `cursor agent`  | `FLEET_CURSOR_CMD`   |
| OpenCode    | `opencode`      | `FLEET_OPENCODE_CMD` |
| Grok        | `grok`          | `FLEET_GROK_CMD`     |

Session file: `~/.config/agent-fleet/session.json`

## Requirements

- Go 1.22+
- `git` on PATH (for worktrees)
- Agent CLIs installed and on `$PATH`

## License

MIT
