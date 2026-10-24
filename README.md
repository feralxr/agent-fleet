# Agent Fleet

Lightweight **TUI cockpit** for running and switching between multiple coding agents — with isolation, fan-out, and review tools.

Default agents: **Claude Code**, **Cursor**, **OpenCode**, **Grok**.  
Add any CLI via `fleet.toml`.

## Highlights

| Area | Features |
|------|----------|
| **Control** | Start / stop / restart, split view, session restore |
| **Isolation** | Git worktree per agent (`w`) |
| **Parallel** | Fan-out one prompt to all running agents (`f`) |
| **Attention** | Waiting detection + jump with `space` |
| **Review** | `git diff --stat` panel (`g`) |
| **PTY** | Real PTY by default for interactive agent TUIs |
| **Config** | `fleet.toml` + env overrides + custom profiles |
| **Ship** | Tests, Makefile, GitHub Actions multi-arch releases |

## Install

```bash
go install github.com/feralxr/agent-fleet/cmd/fleet@latest
```

From source:

```bash
git clone https://github.com/feralxr/agent-fleet.git
cd agent-fleet
make build    # or: go build -o fleet ./cmd/fleet
```

## Demo

See [docs/demo.md](docs/demo.md) for a walkthrough of the main layout and flow.

```text
AGENTS            live output / diff
● running         › your prompt
◐ needs you       agent response…
```

## Quick start

```bash
cd /path/to/your/repo
fleet
```

```text
w          create worktree for selected agent
s          start agent (PTY mode by default)
f          fan-out mode → type prompt → Enter
space      jump to agent that needs input
g          toggle git diff view
v          split view
q          quit (saves session)
```

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Previous agent |
| `↓` / `j` | Next agent |
| `s` | Start |
| `x` | Stop |
| `r` | Restart |
| `space` | Jump to next agent needing attention |
| `w` | Create / attach git worktree |
| `f` | Fan-out prompt to all running agents |
| `g` | Toggle git diff --stat |
| `ctrl+r` | Reload fleet.toml |
| `S` | Save named session |
| `L` | Load next named session |
| `↑`/`↓` (input) | Prompt history |
| `d` | Change working directory |
| `v` | Toggle split view |
| `i` | Focus prompt input |
| `Esc` | Cancel / unfocus |
| `Enter` | Send / confirm |
| `c` | Copy output |
| `ctrl+l` | Clear output |
| `?` | Help |
| `q` | Quit |

## Configuration (`fleet.toml`)

Search order:

1. `$FLEET_CONFIG`
2. `./fleet.toml`
3. `~/.config/agent-fleet/fleet.toml`

Generate an example:

```bash
fleet --init-config
# or: make example
```

### Add custom agents

```toml
use_pty = true

[[extra_agents]]
id = "aider"
name = "Aider"
command = "aider"
args = ["--no-git"]
use_pty = true

[[extra_agents]]
id = "codex"
name = "Codex"
command = "codex"
use_pty = true
```

### Replace built-ins

Use `[[agents]]` instead of `[[extra_agents]]` to define the full list yourself.

### Env overrides

| Variable | Effect |
|----------|--------|
| `FLEET_CLAUDE_CMD` | Override Claude binary |
| `FLEET_CURSOR_CMD` | Override Cursor binary |
| `FLEET_OPENCODE_CMD` | Override OpenCode |
| `FLEET_GROK_CMD` | Override Grok |
| `FLEET_<ID>_CMD` | Override any custom agent id |

### PTY vs pipes

- `use_pty = true` (default for built-ins): better interactive TUI compatibility; ANSI stripped for the log pane.
- `use_pty = false`: classic stdin/stdout pipes.

## Worktrees

`w` creates:

```text
<repo>/.fleet/worktrees/<agent-id>
```

on branch `fleet/<agent-id>`. The agent’s process cwd is set to that path. Add `.fleet/` to `.gitignore` if you want worktrees local-only.

## Session

Saved on quit to `~/.config/agent-fleet/session.json`:

- Last cwd
- Last active agent
- Worktree map

## Development

```bash
make test
make build
make release   # cross-compile to dist/
```

CI runs tests on every push; tagged releases (`v*`) publish linux/darwin/windows binaries + checksums.

```bash
fleet --version
```

## Requirements

- Go 1.22+
- `git` (for worktrees / diff)
- Agent CLIs on `$PATH`

## License

MIT
