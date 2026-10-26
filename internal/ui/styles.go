package ui

import "github.com/charmbracelet/lipgloss"

// Styles — dark violet cockpit palette
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#6D28D9")).
			Padding(0, 1)

	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E9D5FF")).
			Background(lipgloss.Color("#3B0764")).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))

	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F5F3FF"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#C4B5FD")).
			Background(lipgloss.Color("#16121F")).
			Padding(0, 1)

	leftStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#3F3F46")).
			Padding(0, 1, 0, 0)

	rightStyle = lipgloss.NewStyle().
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A1A1AA")).
			Background(lipgloss.Color("#09090B"))

	statusIdle = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
	statusRun  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ADE80"))
	statusWait = lipgloss.NewStyle().Foreground(lipgloss.Color("#38BDF8"))
	statusErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	statusMiss = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))

	fleetLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B")).Italic(true)
	promptLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD")).Bold(true)

	activeBar   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	selectedRow = lipgloss.NewStyle().Background(lipgloss.Color("#1A1229"))
)
