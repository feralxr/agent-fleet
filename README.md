# Agent Fleet

Lightweight terminal UI for running and switching between multiple coding agents side-by-side.

Supports **Claude Code**, **Cursor**, **OpenCode**, and **Grok**.

## Features

- Launch / stop / restart any agent
- Switch active agent instantly
- **Split view** — watch two agents side-by-side
- Change working directory from inside the TUI
- Shared prompt input
- Live output with markdown rendering
- Session persistence (last cwd + active agent)
- Copy output to clipboard
- Binary detection (shows missing agents clearly)
- Keyboard-first interface

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
| `d`        | Change working directory      |
| `v`        | Toggle split view             |
| `i`        | Focus prompt input            |
| `Esc`      | Cancel / unfocus              |
| `Enter`    | Send prompt / confirm path    |
| `c`        | Copy output to clipboard      |
| `ctrl+l`   | Clear output                  |
| `?`        | Toggle help                   |
| `q`        | Quit (saves session)          |

### Split view

Press `v` to show the selected agent and a neighbor side-by-side. Press `v` again to return to single pane.

### Change directory

Press `d`, type a path, press Enter. Invalid paths are rejected. The new cwd is saved in the session.

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
- Agent CLIs installed and on `$PATH`

## License

MIT
