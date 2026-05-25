package ui

import "github.com/charmbracelet/lipgloss"

// Palette — sober, terminal-native. Chosen to read well on both light and
// dark backgrounds without setting bg on every line.
var (
	colorFg      = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#94A3B8"}
	colorSubtle  = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#64748B"}
	colorBorder  = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#334155"}
	colorBorderH = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#475569"}

	colorAccent  = lipgloss.Color("#8B5CF6") // violet — brand
	colorAccent2 = lipgloss.Color("#A78BFA") // violet light
	colorAI      = lipgloss.Color("#06B6D4") // cyan — AI assistant
	colorSafe    = lipgloss.Color("#10B981") // green
	colorMed     = lipgloss.Color("#F59E0B") // amber
	colorDang    = lipgloss.Color("#EF4444") // red
	colorSel     = lipgloss.Color("#3B82F6") // blue — selection
	colorInk     = lipgloss.Color("#FFFFFF")
)

// Chrome — header band, tabs, info strip, footer.
var (
	brandStyle = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorAccent).
			Bold(true).
			Padding(0, 2)

	brandVersionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#DDD6FE")).
				Background(colorAccent).
				Padding(0, 1)

	viewTitleStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true)

	viewSubtitleStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	diskInfoStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	tabActive = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorAccent).
			Bold(true).
			Padding(0, 2)

	tabInactive = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2)

	tabSeparator = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Render("·")

	infoStripStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			PaddingLeft(1)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Background(lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#0F172A"}).
			Padding(0, 1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent2).
			Bold(true)

	footerSepStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	hrule = lipgloss.NewStyle().Foreground(colorBorder)
)

// Body panels — bordered + titled.
var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	panelDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderH).
			Padding(1, 2)

	sectionStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			MarginTop(1)
)

// Lists, rows, search & modal.
var (
	headerStyle = lipgloss.NewStyle().Foreground(colorFg).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	subtleStyle = lipgloss.NewStyle().Foreground(colorSubtle)
	rowStyle    = lipgloss.NewStyle().Foreground(colorFg)
	dimStyle    = lipgloss.NewStyle().Foreground(colorMuted)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(colorFg).
				Background(lipgloss.AdaptiveColor{Light: "#EEF2FF", Dark: "#1E293B"}).
				Bold(true)

	markedStyle = lipgloss.NewStyle().Foreground(colorSel).Bold(true)

	riskSafe = lipgloss.NewStyle().Foreground(colorSafe).Bold(true)
	riskMed  = lipgloss.NewStyle().Foreground(colorMed).Bold(true)
	riskDang = lipgloss.NewStyle().Foreground(colorDang).Bold(true)

	riskChipSafe = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorSafe).
			Bold(true).
			Padding(0, 1)
	riskChipMed = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorMed).
			Bold(true).
			Padding(0, 1)
	riskChipDang = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorDang).
			Bold(true).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorAccent).
			Padding(1, 3)
)

// AI chat — role badges, input boxes, sample-prompt chips.
var (
	aiBadgeStyle = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorAI).
			Bold(true).
			Padding(0, 1)

	userBadgeStyle = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorAccent).
			Bold(true).
			Padding(0, 1)

	systemBadgeStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true)

	chipStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderH).
			Padding(0, 1).
			MarginRight(1)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderH).
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

func riskChip(label string) string {
	switch label {
	case "SAFE":
		return riskChipSafe.Render(" SAFE ")
	case "MEDIUM":
		return riskChipMed.Render(" MED ")
	case "DANGEROUS":
		return riskChipDang.Render(" DANGER ")
	}
	return mutedStyle.Render(label)
}

// footerHint renders "key act" chunks separated by a thin dot.
type hint struct{ key, label string }

func renderHints(hints []hint) string {
	var parts []string
	for i, h := range hints {
		parts = append(parts, footerKeyStyle.Render(h.key)+" "+h.label)
		_ = i
	}
	sep := " " + footerSepStyle.Render("·") + " "
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
