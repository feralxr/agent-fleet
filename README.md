# Agent Fleet

Lightweight terminal UI for running and switching between multiple coding agents side-by-side.

Supports:

- **Claude Code**
- **Cursor**
- **OpenCode**
- **Grok**

## Features

- Launch / stop / restart any agent with a single key
- Quickly switch the active agent
- Shared working directory
- Live output view with markdown rendering
- Shared prompt input sent to the active agent
- Status indicators (idle / running / error / missing)
- Session persistence (remembers last cwd + active agent)
- Copy output to clipboard
- Keyboard-first interface
- Clean multi-pane layout

## Install

```bash
go install github.com/feralxr/agent-fleet/cmd/fleet@latest
```

Or build from source:

```bash
git clone https://github.com/feralxr/agent-fleet.git
cd agent-fleet
go build -o fleet ./cmd/fleet
```

## Usage

```bash
fleet
```

### Keybindings

| Key        | Action                        |
|------------|-------------------------------|
| `↑` / `k`  | Previous agent                |
| `↓` / `j`  | Next agent                    |
| `s`        | Start selected agent          |
| `x`        | Stop selected agent           |
| `r`        | Restart selected agent        |
| `i`        | Focus prompt input            |
| `Esc`      | Unfocus prompt input          |
| `Enter`    | Send prompt to active agent   |
| `c`        | Copy output to clipboard      |
| `ctrl+l`   | Clear output                  |
| `?`        | Toggle help                   |
| `q`        | Quit (saves session)          |

## Configuration

### Command overrides

| Agent       | Default command | Override env var     |
|-------------|-----------------|----------------------|
| Claude Code | `claude`        | `FLEET_CLAUDE_CMD`   |
| Cursor      | `cursor agent`  | `FLEET_CURSOR_CMD`   |
| OpenCode    | `opencode`      | `FLEET_OPENCODE_CMD` |
| Grok        | `grok`          | `FLEET_GROK_CMD`     |

```bash
export FLEET_CLAUDE_CMD="claude"
export FLEET_CURSOR_CMD="cursor"
fleet
```

### Session

On quit, Agent Fleet saves:

- Last working directory
- Last active agent

to `~/.config/agent-fleet/session.json` and restores them on the next launch.

## Requirements

- Go 1.22+
- The agent CLIs you want to use must be installed and available in `$PATH`

## License

MIT
