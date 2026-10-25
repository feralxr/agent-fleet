# Agent Fleet

[![CI](https://github.com/feralxr/agent-fleet/actions/workflows/ci.yml/badge.svg)](https://github.com/feralxr/agent-fleet/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/feralxr/agent-fleet)](https://github.com/feralxr/agent-fleet/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Lightweight **TUI cockpit** for running and switching between multiple coding agents — with isolation, fan-out, and review tools.

Default agents: **Claude Code**, **Cursor**, **OpenCode**, **Grok**.  
Add any CLI via `fleet.toml`.

## Highlights

| Area | Features |
|------|----------|
| **Control** | Start / stop / restart / stop-all, split view, session restore |
| **Isolation** | Git worktree per agent (`w`) |
| **Parallel** | Fan-out one prompt to all running agents (`f`) |
| **Attention** | Waiting detection + jump with `space` |
| **Review** | `git diff --stat` panel (`g`) |
| **PTY** | Real PTY by default for interactive agent TUIs |
| **Config** | `fleet.toml` + env overrides + custom profiles |
| **UX** | Fast redraws, mouse scroll, 1–9 agent jump, prompt history |

## Install

**From GitHub Releases** (recommended):

https://github.com/feralxr/agent-fleet/releases/latest

Binaries: Linux/macOS (`x64`, `arm64`), Windows (`x64`, `x86`).

**Go:**

```bash
go install github.com/feralxr/agent-fleet/cmd/fleet@latest
```

**From source:**

```bash
git clone https://github.com/feralxr/agent-fleet.git
cd agent-fleet
make build    # or: go build -o fleet ./cmd/fleet
```

## Quick start

```bash
cd /path/to/your/repo
fleet
```

```text
1-9 / j/k  select agent
w          create worktree
s          start (PTY by default)
i          focus prompt → type → Enter
f          fan-out to all running agents
space      jump to agent that needs input
g          toggle git diff view
v          split view
q          quit (saves session)
```

See [docs/demo.md](docs/demo.md) for a layout walkthrough.

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` · `↓` / `j` | Previous / next agent (wraps) |
| `1`–`9` | Select agent by index |
| `s` | Start |
| `x` | Stop |
| `X` | Stop all |
| `r` | Restart |
| `space` | Jump to next agent needing attention |
| `w` | Create / attach git worktree |
| `f` | Fan-out prompt to all running |
| `g` | Toggle git diff --stat |
| `d` | Change working directory |
| `v` | Toggle split view |
| `i` | Focus prompt input |
| `Esc` | Cancel / unfocus |
| `Enter` | Send / confirm |
| `↑` / `↓` (input) | Prompt history |
| `pgup` / `pgdn` | Scroll output |
| `ctrl+u` / `ctrl+d` | Half-page scroll |
| Mouse wheel | Scroll output |
| `c` | Copy output |
| `ctrl+l` | Clear output |
| `ctrl+r` | Reload `fleet.toml` |
| `S` / `L` | Save / load named session |
| `?` | Help |
| `q` | Quit |

## Configuration (`fleet.toml`)

Search order:

1. `$FLEET_CONFIG`
2. `./fleet.toml`
3. `~/.config/agent-fleet/fleet.toml`

```bash
fleet --init-config
# or: make example
```

```toml
use_pty = true

[[extra_agents]]
id = "aider"
name = "Aider"
command = "aider"
args = ["--no-git"]
use_pty = true
```

Use `[[agents]]` instead of `[[extra_agents]]` to replace built-ins entirely.

| Variable | Effect |
|----------|--------|
| `FLEET_CLAUDE_CMD` | Override Claude binary |
| `FLEET_CURSOR_CMD` | Override Cursor |
| `FLEET_OPENCODE_CMD` | Override OpenCode |
| `FLEET_GROK_CMD` | Override Grok |
| `FLEET_<ID>_CMD` | Override any custom agent id |

## Worktrees & sessions

- `w` → `<repo>/.fleet/worktrees/<agent-id>` on branch `fleet/<agent-id>`
- Add `.fleet/` to the project `.gitignore` if worktrees should stay local
- Session file: `~/.config/agent-fleet/session.json` (cwd, active agent, worktrees, prompt history, named sessions)

## Development

```bash
make test
make build
make release
fleet --version
```

See [CONTRIBUTING.md](CONTRIBUTING.md). CI tests every push; tags `v*` publish multi-arch binaries.

## Requirements

- Go 1.22+
- `git` (worktrees / diff)
- Agent CLIs on `$PATH`

## License

MIT — see [LICENSE](LICENSE)
