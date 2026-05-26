package ui

import "github.com/charmbracelet/lipgloss"

// Palette — gold brand, terminal-native. Chosen to read well on both light and
// dark backgrounds without setting bg on every line.
var (
	colorFg      = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	colorMuted   = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#94A3B8"}
	colorSubtle  = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#64748B"}
	colorBorder  = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#334155"}
	colorBorderH = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#475569"}

	colorAccent  = lipgloss.Color("#D4A017") // pure gold — brand
	colorAccent2 = lipgloss.Color("#F5C842") // gold light
	colorAccent3 = lipgloss.Color("#B8860B") // dark gold
	colorAI      = lipgloss.Color("#B8860B") // dark gold — AI assistant
	colorSafe    = lipgloss.Color("#10B981") // green
	colorMed     = lipgloss.Color("#F59E0B") // amber
	colorDang    = lipgloss.Color("#EF4444") // red
	colorSel     = lipgloss.Color("#3B82F6") // blue — selection
	colorInk     = lipgloss.Color("#0A0A0A") // near-black ink on gold
	colorGoldDim = lipgloss.Color("#7A5C00") // dim gold for subtle use

	// AI process step colors (cohesive professional theme)
	colorAIThink  = lipgloss.Color("#D4A017") // pure gold
	colorAITool   = lipgloss.Color("#F5C842") // light gold
	colorAISystem = lipgloss.Color("#9CA3AF") // subtle slate
	colorAIExec   = lipgloss.Color("#F59E0B") // amber for executing
)

// Chrome — header band, tabs, info strip, footer.
var (
	brandStyle = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorAccent).
			Bold(true).
			Padding(0, 2)

	brandVersionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3D2800")).
				Background(colorAccent3).
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
			BorderForeground(colorBorder).
			Padding(1, 4)

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
				Foreground(colorInk).
				Background(colorAccent).
				Bold(true)

	markedStyle = lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)

	riskSafe = lipgloss.NewStyle().Foreground(colorSafe).Bold(true)
	riskMed  = lipgloss.NewStyle().Foreground(colorMed).Bold(true)
	riskDang = lipgloss.NewStyle().Foreground(colorDang).Bold(true)

	riskChipSafe = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorSafe).
			Bold(true).
			Padding(0, 1)
	riskChipMed = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorMed).
			Bold(true).
			Padding(0, 1)
	riskChipDang = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorDang).
			Bold(true).
			Padding(0, 1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorAccent).
			Padding(1, 3)

	// Search highlight
	searchBannerStyle = lipgloss.NewStyle().
				Foreground(colorInk).
				Background(colorAccent).
				Bold(true).
				Padding(0, 1)

	searchTermStyle = lipgloss.NewStyle().
			Foreground(colorAccent2).
			Bold(true)

	// Scroll position indicator
	scrollStyle = lipgloss.NewStyle().
			Foreground(colorGoldDim).
			Italic(true)
)

// AI chat — role badges, input boxes, sample-prompt chips, process steps.
var (
	aiBadgeStyle = lipgloss.NewStyle().
			Foreground(colorAI).
			Bold(true)

	userBadgeStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	systemBadgeStyle = lipgloss.NewStyle().
				Foreground(colorAISystem).
				Bold(true)

	// Process step badges
	thinkBadgeStyle = lipgloss.NewStyle().
			Foreground(colorAIThink).
			Bold(true)

	toolBadgeStyle = lipgloss.NewStyle().
			Foreground(colorAITool).
			Bold(true)

	execBadgeStyle = lipgloss.NewStyle().
			Foreground(colorAIExec).
			Bold(true)

	scanBadgeStyle = lipgloss.NewStyle().
			Foreground(colorAccent2).
			Bold(true)

	doneBadgeStyle = lipgloss.NewStyle().
			Foreground(colorSafe).
			Bold(true)

	errorBadgeStyle = lipgloss.NewStyle().
			Foreground(colorDang).
			Bold(true)

	chipStyle = lipgloss.NewStyle().
			Foreground(colorAccent2).
			Italic(true)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorderH).
			Padding(0, 1)

	inputActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)

	// Approval dialog styles
	approvalBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(colorAccent).
				Padding(1, 2)

	approvalSafeStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(colorSafe).
				Padding(1, 2)

	approvalMedStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(colorMed).
				Padding(1, 2)

	approvalDangStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(colorDang).
				Padding(1, 2)

	approveKeyStyle = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorSafe).
			Bold(true).
			Padding(0, 1)

	rejectKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorDang).
			Bold(true).
			Padding(0, 1)

	autoKeyStyle = lipgloss.NewStyle().
			Foreground(colorInk).
			Background(colorAccent2).
			Bold(true).
			Padding(0, 1)

	cancelKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorAISystem).
			Bold(true).
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

// approvalBoxForRisk returns the correct border style for the risk level.
func approvalBoxForRisk(label string) lipgloss.Style {
	switch label {
	case "SAFE":
		return approvalSafeStyle
	case "MEDIUM":
		return approvalMedStyle
	case "DANGEROUS":
		return approvalDangStyle
	}
	return approvalBoxStyle
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
