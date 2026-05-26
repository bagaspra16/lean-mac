package ui

import (
	"fmt"
	"strings"

	"github.com/bagaspra16/lean-mac/internal/ai"
	"github.com/bagaspra16/lean-mac/internal/config"
	"github.com/bagaspra16/lean-mac/internal/fsutil"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

// Version is set via -ldflags at build time. Shown in the brand chip.
var Version = "dev"

type tabID int

const (
	tabScan tabID = iota
	tabAI
	tabHelp
)

// view contract — each child renders its body and tells chrome what to show.
type view interface {
	Init() tea.Cmd
	Update(tea.Msg) (view, tea.Cmd)
	View(width, bodyHeight int) string
	Title() string    // bold title shown in header row 2
	Subtitle() string // muted descriptor next to title (e.g. "what this view does")
	Status() string   // right-aligned status, e.g. "47 findings · 16 GiB"
	Footer() []hint   // grouped key hints for the bottom bar
}

// App is the root model.
type App struct {
	cfg       config.Config
	tab       tabID
	width     int
	height    int
	scan      view
	ai        view
	help      view
	diskFree  int64
	diskTotal int64
}

func NewApp(cfg config.Config, dryRun bool) App {
	free, total, _ := fsutil.DiskUsage("/")
	app := App{
		cfg:       cfg,
		tab:       tabScan,
		diskFree:  free,
		diskTotal: total,
		scan:      newScanView(dryRun),
		help:      newHelpView(cfg.HasAI()),
	}
	if cfg.HasAI() {
		client := ai.NewClient(cfg.GroqKeys, cfg.Model)
		app.ai = newAIView(client, dryRun)
	} else {
		app.ai = newAIDisabledView()
	}
	return app
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.scan.Init(), a.ai.Init(), a.help.Init())
}

func (a App) active() view {
	switch a.tab {
	case tabAI:
		return a.ai
	case tabHelp:
		return a.help
	default:
		return a.scan
	}
}

func (a App) setActive(v view) App {
	switch a.tab {
	case tabAI:
		a.ai = v
	case tabHelp:
		a.help = v
	default:
		a.scan = v
	}
	return a
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		v, cmd1 := a.scan.Update(msg)
		a.scan = v
		v, cmd2 := a.ai.Update(msg)
		a.ai = v
		v, cmd3 := a.help.Update(msg)
		a.help = v
		return a, tea.Batch(cmd1, cmd2, cmd3)
	case tickMsg:
		v, cmd1 := a.scan.Update(msg)
		a.scan = v
		v, cmd2 := a.ai.Update(msg)
		a.ai = v
		return a, tea.Batch(cmd1, cmd2)
	case tea.KeyMsg:
		if !a.activeWantsKey(msg) {
			switch msg.String() {
			case "ctrl+c":
				return a, tea.Quit
			case "tab":
				a.tab = (a.tab + 1) % 3
				return a, nil
			case "shift+tab":
				a.tab = (a.tab + 2) % 3
				return a, nil
			case "1":
				a.tab = tabScan
				return a, nil
			case "2":
				a.tab = tabAI
				return a, nil
			case "3":
				a.tab = tabHelp
				return a, nil
			case "?":
				a.tab = tabHelp
				return a, nil
			case "q":
				return a, tea.Quit
			}
		}
	}
	v, cmd := a.active().Update(msg)
	return a.setActive(v), cmd
}

func (a App) activeWantsKey(msg tea.KeyMsg) bool {
	if a.tab != tabAI {
		return false
	}
	v, ok := a.ai.(*aiView)
	if !ok {
		return false
	}
	return v.wantsKey(msg)
}

// --- Layout ---
//
//   row 0   brand band                 disk stats
//   row 1   view title  · subtitle              view status
//   row 2   tab bar
//   row 3   info strip (one-liner about this view)
//   row 4   ─── horizontal rule ───
//   rows…   body (height-7 rows)
//   row n-1 footer key hints
//
// Six rows of chrome total.

const chromeRows = 7

func (a App) View() string {
	if a.width == 0 {
		return "initializing…"
	}
	bodyHeight := a.height - chromeRows
	if bodyHeight < 5 {
		bodyHeight = 5
	}
	parts := []string{
		a.renderBrandRow(),
		a.renderTitleRow(),
		a.renderTabRow(),
		a.renderInfoStrip(),
		hrule.Render(strings.Repeat("─", a.width)),
		a.active().View(a.width, bodyHeight),
		a.renderFooter(),
	}
	return strings.Join(parts, "\n")
}

func (a App) renderBrandRow() string {
	brand := brandStyle.Render("LEAN-MAC") + brandVersionStyle.Render(Version)
	used := a.diskTotal - a.diskFree
	pct := 0
	if a.diskTotal > 0 {
		pct = int(float64(used) / float64(a.diskTotal) * 100)
	}
	bar := miniBar(pct, 16)
	disk := fmt.Sprintf("%s  %s used of %s  ·  free %s",
		bar,
		humanize.IBytes(uint64(used)),
		humanize.IBytes(uint64(a.diskTotal)),
		humanize.IBytes(uint64(a.diskFree)),
	)
	right := diskInfoStyle.Render(disk)
	return padBetween(brand, right, a.width)
}

func (a App) renderTitleRow() string {
	v := a.active()
	left := viewTitleStyle.Render(v.Title())
	if sub := v.Subtitle(); sub != "" {
		left += "  " + viewSubtitleStyle.Render(sub)
	}
	right := viewSubtitleStyle.Render(v.Status())
	return padBetween(left, right, a.width)
}

func (a App) renderTabRow() string {
	tabs := []struct {
		id    tabID
		label string
	}{
		{tabScan, " 1 · Scan "},
		{tabAI, " 2 · AI Cleanse "},
		{tabHelp, " 3 · Help "},
	}
	var b strings.Builder
	for i, t := range tabs {
		if i > 0 {
			b.WriteString(" ")
			b.WriteString(tabSeparator)
			b.WriteString(" ")
		}
		if t.id == a.tab {
			b.WriteString(tabActive.Render(t.label))
		} else {
			b.WriteString(tabInactive.Render(t.label))
		}
	}
	return b.String()
}

func (a App) renderInfoStrip() string {
	var s string
	switch a.tab {
	case tabScan:
		s = "Browse reclaimable artifacts grouped by category. Mark items, then press d to delete with confirmation."
	case tabAI:
		s = "Chat with the AI to scan and clean interactively. It proposes one category at a time — you approve each step."
	case tabHelp:
		s = "Glossary, keybindings, risk model, and safety guarantees."
	}
	return infoStripStyle.Render(s)
}

func (a App) renderFooter() string {
	body := a.active().Footer()
	global := []hint{
		{"tab", "switch"},
		{"?", "help"},
		{"q", "quit"},
	}
	left := renderHints(body)
	right := renderHints(global)
	full := padBetween(left, right, a.width-2)
	return footerStyle.Width(a.width).Render(full)
}

// padBetween places left at the start, right at the end of a `width`-wide row.
func padBetween(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// miniBar renders an inline disk-usage bar like ▰▰▰▰▰▰▱▱▱▱.
func miniBar(pct, width int) string {
	if width < 1 {
		return ""
	}
	if pct < 0 {
		pct = 0
	} else if pct > 100 {
		pct = 100
	}
	filled := pct * width / 100
	var b strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			b.WriteString("▰")
		} else {
			b.WriteString("▱")
		}
	}
	color := colorSafe
	switch {
	case pct >= 90:
		color = colorDang
	case pct >= 75:
		color = colorMed
	}
	return lipgloss.NewStyle().Foreground(color).Render(b.String())
}
