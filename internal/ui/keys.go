package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Quit         key.Binding
	Help         key.Binding
	Start        key.Binding
	Stop         key.Binding
	StopAll      key.Binding
	Restart      key.Binding
	Clear        key.Binding
	FocusIn      key.Binding
	FocusOut     key.Binding
	Copy         key.Binding
	Cwd          key.Binding
	Split        key.Binding
	Worktree     key.Binding
	Fan          key.Binding
	Jump         key.Binding
	Diff         key.Binding
	Reload       key.Binding
	SaveName     key.Binding
	LoadName     key.Binding
	HistUp       key.Binding
	HistDown     key.Binding
	ModelSel     key.Binding
	Resend       key.Binding
	Export       key.Binding
	ToggleTarget key.Binding
	Templates    key.Binding
	FocusMode    key.Binding
	Note         key.Binding
	Compare      key.Binding
	PinFire      key.Binding
	PgUp         key.Binding
	PgDown       key.Binding
	HalfUp       key.Binding
	HalfDown     key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "prev agent"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "next agent"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send / confirm"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start agent"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop agent"),
		),
		StopAll: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "stop all agents"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart agent"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear output"),
		),
		FocusIn: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "focus input"),
		),
		FocusOut: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel / unfocus"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy output"),
		),
		Cwd: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "change directory"),
		),
		Split: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "toggle split view"),
		),
		Worktree: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "create worktree"),
		),
		Fan: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fan prompt to all"),
		),
		Jump: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "jump to needs attention"),
		),
		Diff: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "toggle git diff"),
		),
		Reload: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "reload fleet.toml"),
		),
		SaveName: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "save named session"),
		),
		LoadName: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "load named session"),
		),
		HistUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "prev prompt (input)"),
		),
		HistDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next prompt (input)"),
		),
		ModelSel: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "select model"),
		),
		Resend: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "resend last prompt"),
		),
		Export: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "export output"),
		),
		ToggleTarget: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle fan-out target"),
		),
		Templates: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "prompt templates"),
		),
		FocusMode: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "focus mode"),
		),
		Note: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "agent note"),
		),
		Compare: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "compare outputs"),
		),
		PinFire: key.NewBinding(
			key.WithKeys("'"),
			key.WithHelp("'", "pinned template"),
		),
		PgUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+b"),
			key.WithHelp("pgup", "scroll up"),
		),
		PgDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("pgdown", "scroll down"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		HalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Start, k.Stop, k.Jump, k.Diff, k.Worktree, k.Fan, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Start, k.Stop, k.StopAll, k.Restart},
		{k.Jump, k.Diff, k.Worktree, k.Fan, k.ModelSel, k.Split},
		{k.Resend, k.Export, k.Copy, k.Clear, k.Cwd, k.Reload},
		{k.PgUp, k.PgDown, k.FocusIn, k.Help, k.Quit},
	}
}

// Model is the main Bubble Tea model.
