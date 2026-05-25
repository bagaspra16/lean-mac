package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bagaspra16/lean-mac/internal/types"

	tea "github.com/charmbracelet/bubbletea"
)

type helpView struct {
	hasAI         bool
	scroll        int
	width, height int
	totalRows     int
}

func newHelpView(hasAI bool) *helpView { return &helpView{hasAI: hasAI} }

func (h *helpView) Init() tea.Cmd { return nil }

func (h *helpView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		h.width, h.height = m.Width, m.Height
	case tea.KeyMsg:
		switch m.String() {
		case "j", "down":
			if h.scroll < h.totalRows-h.height {
				h.scroll++
			}
		case "k", "up":
			if h.scroll > 0 {
				h.scroll--
			}
		case "g":
			h.scroll = 0
		case "G":
			h.scroll = h.totalRows - h.height
			if h.scroll < 0 {
				h.scroll = 0
			}
		}
	}
	return h, nil
}

func (h *helpView) Title() string    { return "Help & Glossary" }
func (h *helpView) Subtitle() string { return "what each thing means, and the rules" }
func (h *helpView) Status() string {
	if h.totalRows > h.height {
		return fmt.Sprintf("scroll j/k · %d/%d", h.scroll, h.totalRows-h.height)
	}
	return ""
}
func (h *helpView) Footer() []hint {
	return []hint{
		{"j/k", "scroll"},
		{"g/G", "top/bottom"},
		{"1", "Scan"},
		{"2", "AI"},
	}
}

func (h *helpView) View(width, height int) string {
	h.width, h.height = width, height
	rows := h.buildRows()
	h.totalRows = len(rows)
	end := h.scroll + height
	if end > len(rows) {
		end = len(rows)
	}
	visible := rows[h.scroll:end]
	return strings.Join(visible, "\n")
}

func (h *helpView) buildRows() []string {
	out := []string{}
	out = append(out, h.sectionWhat()...)
	out = append(out, "")
	out = append(out, h.sectionTabs()...)
	out = append(out, "")
	out = append(out, h.sectionRisks()...)
	out = append(out, "")
	out = append(out, h.sectionGlossary()...)
	out = append(out, "")
	out = append(out, h.sectionSafety()...)
	out = append(out, "")
	out = append(out, h.sectionAI()...)
	out = append(out, "")
	return out
}

func (h *helpView) sectionWhat() []string {
	return []string{
		panelTitleStyle.Render("What is lean-mac"),
		"",
		"  A terminal-native storage tool for macOS developers. It finds where",
		"  your disk is going (Docker, Xcode, node_modules, package caches), shows",
		"  you what's reclaimable, and — only with your confirmation — deletes it.",
		"",
		"  " + dimStyle.Render("Two ways to use it:"),
		"    " + footerKeyStyle.Render("1. Scan") + "          browse + manually mark items to remove.",
		"    " + footerKeyStyle.Render("2. AI Cleanse") + "    chat with an LLM that walks you through it.",
	}
}

func (h *helpView) sectionTabs() []string {
	return []string{
		panelTitleStyle.Render("Keys per view"),
		"",
		sectionStyle.Render("Global"),
		"  " + footerKeyStyle.Render("tab / shift+tab") + "   cycle views",
		"  " + footerKeyStyle.Render("1 / 2 / 3") + "         jump to a view",
		"  " + footerKeyStyle.Render("?") + "                 open this Help",
		"  " + footerKeyStyle.Render("q / ctrl+c") + "        quit",
		"",
		sectionStyle.Render("Scan view"),
		"  " + footerKeyStyle.Render("j / k") + "       move cursor",
		"  " + footerKeyStyle.Render("g / G") + "       top / bottom",
		"  " + footerKeyStyle.Render("space") + "       mark item under cursor",
		"  " + footerKeyStyle.Render("a") + "           mark all SAFE items",
		"  " + footerKeyStyle.Render("/") + "           filter (enter/esc to leave)",
		"  " + footerKeyStyle.Render("d") + "           delete marked (asks to confirm)",
		"",
		sectionStyle.Render("AI Cleanse"),
		"  type your question, " + footerKeyStyle.Render("enter") + " to send",
		"  " + footerKeyStyle.Render("y") + " / " + footerKeyStyle.Render("n") + "       approve or reject the current proposal",
		"  " + footerKeyStyle.Render("a") + "           auto-approve future SAFE proposals only",
		"  " + footerKeyStyle.Render("c") + "           cancel the agent",
	}
}

func (h *helpView) sectionRisks() []string {
	return []string{
		panelTitleStyle.Render("Risk tiers"),
		"",
		"  " + riskChip("SAFE") + "       " + riskBlurb("SAFE"),
		"             " + dimStyle.Render("Example: npm cache, Homebrew cache, Xcode DerivedData."),
		"",
		"  " + riskChip("MEDIUM") + "     " + riskBlurb("MEDIUM"),
		"             " + dimStyle.Render("Example: Go module cache, iOS simulators, cargo registry."),
		"",
		"  " + riskChip("DANGEROUS") + "  " + riskBlurb("DANGEROUS"),
		"             " + dimStyle.Render("Example: Docker volumes (may hold DBs), Xcode archives."),
	}
}

func (h *helpView) sectionGlossary() []string {
	out := []string{
		panelTitleStyle.Render("Category glossary"),
		"",
	}
	cats := make([]types.Category, 0, len(categoryGlossary))
	for c := range categoryGlossary {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool { return string(cats[i]) < string(cats[j]) })
	for _, c := range cats {
		out = append(out, fmt.Sprintf("  %-22s %s", headerStyle.Render(string(c)), dimStyle.Render(categoryGlossary[c])))
	}
	return out
}

func (h *helpView) sectionSafety() []string {
	return []string{
		panelTitleStyle.Render("Safety guarantees"),
		"",
		"  • Dry-run is " + headerStyle.Render("ON by default") + " in the TUI. Nothing is deleted unless you",
		"    pressed y on a confirmation, and even then only in live mode.",
		"  • The cleaner refuses to touch " + headerStyle.Render("/, /System, /Library, /usr, /bin,"),
		"    " + headerStyle.Render("/sbin, /etc, /var, /Applications, /Users") + ", or your home directory",
		"    itself — at the syscall, before any os.RemoveAll runs.",
		"  • The AI " + headerStyle.Render("cannot specify file paths") + ". Its tool only takes a category",
		"    name from a fixed enum, validated against an allowlist.",
		"  • Docker objects go through " + headerStyle.Render("docker prune") + ", iOS simulators through",
		"    " + headerStyle.Render("xcrun simctl delete") + " — same tools you'd run by hand.",
	}
}

func (h *helpView) sectionAI() []string {
	status := riskSafe.Render("AI configured ✓")
	if !h.hasAI {
		status = riskMed.Render("AI not configured — open the AI tab for setup instructions")
	}
	return []string{
		panelTitleStyle.Render("AI status"),
		"",
		"  " + status,
		"",
		"  Keys are read from, in order:",
		"    " + dimStyle.Render("env vars  GROQ_API_KEY, GROQ_API_KEY_2 … GROQ_API_KEY_9"),
		"    " + dimStyle.Render("config    ~/.config/lean-mac/config.toml"),
		"  When more than one key is configured, lean-mac rotates between them on",
		"  rate-limit (429) errors. Keys are never logged or sent anywhere except",
		"  to api.groq.com over HTTPS.",
	}
}
