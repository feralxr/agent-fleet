# Stability checklist (v0.3.0)

Use this before treating a build as daily-driver ready.

## Install & launch
- [ ] `go build -o fleet ./cmd/fleet` succeeds
- [ ] `./fleet` opens TUI without panic
- [ ] Resize terminal — layout reflows; no freeze

## Agents
- [ ] Select agent with `j/k` or `1`–`9`
- [ ] `s` starts agent (or clear missing-binary message)
- [ ] `x` stops; `r` restarts; `X` stops all
- [ ] Output streams without multi-second key lag

## Models
- [ ] `m` opens picker immediately (defaults)
- [ ] Live list fills in when CLI supports `models`
- [ ] Choice applies on next `s`; running agent gets `/model` when possible

## Prompts & templates
- [ ] `i` → type → Enter sends to active agent
- [ ] `/review` expands before send
- [ ] `T` picker inserts template; pin `1`–`4`; `'` + slot inserts pin
- [ ] `p` resends last prompt to a running agent

## Parallel / isolation
- [ ] `w` creates worktree when in a git repo
- [ ] `t` toggles fan targets; `f` fans to targets (or all running)
- [ ] `space` jumps to agent needing attention

## Session
- [ ] `n` saves a note; survives quit/reopen
- [ ] Model + pins survive quit/reopen
- [ ] `q` stops agents and exits cleanly

## Palette
- [ ] `:` opens; filter `mod`; Enter runs model picker
- [ ] Esc closes without side effects

## Known soft spots
- Claude has no live `models` list — curated aliases only
- Attention heuristics are best-effort (agent UIs differ)
- Windows PTY behavior depends on host terminal
