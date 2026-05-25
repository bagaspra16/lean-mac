package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorFg     = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#94A3B8"}
	colorAccent = lipgloss.Color("#7C3AED")
	colorSafe   = lipgloss.Color("#10B981")
	colorMed    = lipgloss.Color("#F59E0B")
	colorDang   = lipgloss.Color("#EF4444")
	colorSel    = lipgloss.Color("#3B82F6")
	colorAI     = lipgloss.Color("#06B6D4")
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorAccent).
			Bold(true).
			Padding(0, 1)

	tabActive = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true).
			Padding(0, 2).
			Border(lipgloss.Border{Bottom: "▔"}, false, false, true, false).
			BorderForeground(colorAccent)

	tabInactive = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2)

	headerStyle = lipgloss.NewStyle().Foreground(colorFg).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	rowStyle    = lipgloss.NewStyle().Foreground(colorFg)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(lipgloss.AdaptiveColor{Light: "#E0E7FF", Dark: "#1E293B"})

	markedStyle = lipgloss.NewStyle().Foreground(colorSel).Bold(true)

	riskSafe = lipgloss.NewStyle().Foreground(colorSafe).Bold(true)
	riskMed  = lipgloss.NewStyle().Foreground(colorMed).Bold(true)
	riskDang = lipgloss.NewStyle().Foreground(colorDang).Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#111827"}).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	aiBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorAI).
			Bold(true).
			Padding(0, 1)

	userBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorAccent).
			Bold(true).
			Padding(0, 1)

	systemBadgeStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	inputActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)
)

func riskStyle(label string) lipgloss.Style {
	switch label {
	case "SAFE":
		return riskSafe
	case "MEDIUM":
		return riskMed
	case "DANGEROUS":
		return riskDang
	}
	return mutedStyle
}
