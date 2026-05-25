package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type helpView struct {
	hasAI         bool
	width, height int
}

func newHelpView(hasAI bool) *helpView { return &helpView{hasAI: hasAI} }

func (h *helpView) Init() tea.Cmd { return nil }
func (h *helpView) Update(msg tea.Msg) (view, tea.Cmd) {
	if m, ok := msg.(tea.WindowSizeMsg); ok {
		h.width, h.height = m.Width, m.Height
	}
	return h, nil
}
func (h *helpView) Title() string  { return "Help" }
func (h *helpView) Status() string { return "" }
func (h *helpView) Footer() string { return "press 1, 2, 3, or tab to switch views" }

func (h *helpView) View(width, height int) string {
	aiNote := riskSafe.Render("AI configured ✓")
	if !h.hasAI {
		aiNote = riskMed.Render("AI not configured — set GROQ_API_KEY or write ~/.config/lean-mac/config.toml")
	}
	lines := []string{
		headerStyle.Render("Tabs"),
		"  1   Scan       Findings list, mark + delete with confirmation",
		"  2   AI Cleanse Conversational cleanup with per-action approval",
		"  3   Help       This screen",
		"",
		headerStyle.Render("Global"),
		"  tab / shift+tab  switch view",
		"  q / ctrl+c       quit",
		"",
		headerStyle.Render("Scan view"),
		"  j / k        move cursor",
		"  g / G        top / bottom",
		"  space        mark item",
		"  a            mark all SAFE",
		"  /            search; enter or esc to leave",
		"  d            delete marked (asks for confirmation)",
		"",
		headerStyle.Render("AI Cleanse"),
		"  type a message, enter to send",
		"  the AI will scan and propose deletions one by one",
		"  y / n        approve or reject current proposal",
		"  a            auto-approve remaining SAFE proposals only",
		"  c            cancel; ask the AI to stop",
		"",
		headerStyle.Render("Safety"),
		"  · Dry-run mode is on by default in the TUI: no files are actually deleted.",
		"  · The cleaner refuses to touch /, /System, /Library, /usr, /bin, /etc,",
		"    /var, /Applications, /Users, or your home directory.",
		"  · The AI cannot specify file paths — only category names from the scan.",
		"",
		aiNote,
	}
	return strings.Join(lines, "\n")
}
