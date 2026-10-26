# Changelog

## [v0.3.0] — 2026-10-26

Stability-focused release: richer agentic UX, cleaner structure, safer session persistence.

### Features
- **Command palette** (`:`) — filter and run start/stop/model/templates/worktree/fan-out/…
- **Per-agent models** (`m`) — curated catalogs + live CLI discovery (async) + mid-session `/model`
- **Prompt templates** (`T` or `/review`, `/tests`, `/explain`, …)
- **Pinned templates** — pin in `T` with `1`–`4`; fire with `'` then `1`–`4`
- **Per-agent notes** (`n`) — persisted in session
- **Compare mode** (`o`) — split selected + neighbor outputs
- **Focus mode** (`z`) — full-width output, hide sidebar
- **Fan-out targets** (`t`) — send only to checked agents
- **Resend / export** (`p` / `e`)
- **Activity labels + spinner** — streaming / working / quiet / needs you
- **Smarter waiting detection** — permission prompts, y/n, approval language

### Hardening
- Session persist on quit: models, notes, pins, worktrees, cwd
- Palette actions call manager methods directly (no synthetic keys)
- Output snapshot isolation; lifecycle + session roundtrip tests
- UI split: `model.go` / `modes.go` / `view.go` / `palette.go` / `keys.go` / `styles.go`

### Performance (carried from 0.2.x)
- No git/Glamour on hot path; output fingerprinting; smart ticks

## [v0.2.0] — earlier

- Multi-arch releases (Linux/macOS/Windows)
- Worktrees, fan-out, diff panel, PTY, custom agents
- Brand assets + README

## [v0.1.x] — earlier

- Initial TUI cockpit for Claude / Cursor / OpenCode / Grok
