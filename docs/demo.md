# Agent Fleet — demo walkthrough

```text
┌ Agent Fleet ─────────────────────────────────────────────┐
│ AGENTS              │  [fleet] started Claude (pty)      │
│ ▌ ● Claude Code     │  › refactor auth to JWT            │
│     wt:claude · run │  ... agent output ...              │
│   ◐ Cursor          │                                    │
│     needs you       │                                    │
│   ● OpenCode        │                                    │
│     idle · pty      │                                    │
├─────────────────────┴────────────────────────────────────┤
│ Type a prompt and press Enter…                           │
├──────────────────────────────────────────────────────────┤
│ prompt sent  ·  2 run · 1 need you  ·  Claude  ·  ~/proj │
│ ? help · space jump · g diff · S save · L load · q quit  │
└──────────────────────────────────────────────────────────┘
```

## Typical flow

1. `cd` into a git repo, run `fleet`
2. Select Claude → `w` (worktree) → `s` (start)
3. Select Cursor → `w` → `s`
4. `f` → type task → Enter (fan-out to both)
5. `space` jumps to whoever asks for approval
6. `g` reviews `git diff --stat` per agent
7. `S` saves a named session; `L` cycles restored layouts
8. In input focus, `↑` / `↓` walk prompt history
