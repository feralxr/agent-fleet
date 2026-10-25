# Agent Fleet — demo walkthrough

```text
┌ ✦ Agent Fleet ── input ── ● Claude Code ─────────────────┐
│ AGENTS              │ OUTPUT · Claude Code · end         │
│ ▌ ● 1 Claude Code   │ ── Claude Code · up 3m12s · 40 ──  │
│     wt:claude · 3m  │ › refactor auth to JWT             │
│   ◉ 2 Cursor        │ … agent stream …                   │
│     needs you       │                                    │
│   ○ 3 OpenCode      │                                    │
│     opencode · pty  │                                    │
│                     │                                    │
│   ● run  ◉ wait     │                                    │
│   1-9 select        │                                    │
├─────────────────────┴────────────────────────────────────┤
│ input  Type a prompt — Enter to send…                    │
├──────────────────────────────────────────────────────────┤
│ started Claude          2 run · 1 need you  ~/proj  15:04│
│ ? help · j/k or 1-9 · s start · space jump · i prompt    │
└──────────────────────────────────────────────────────────┘
```

## Typical flow

1. `cd` into a git repo → run `fleet`
2. Select agent (`j/k` or `1-9`) → `w` worktree → `s` start
3. `f` → type task → Enter (fan-out)
4. `space` jumps to whoever needs approval
5. `g` reviews `git diff --stat`
6. Mouse wheel or `pgup`/`pgdn` to scroll output
7. `S` saves a named session; `L` restores
8. In input focus, `↑`/`↓` walk prompt history
