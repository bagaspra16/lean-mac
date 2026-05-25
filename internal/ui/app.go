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

type tabID int

const (
	tabScan tabID = iota
	tabAI
	tabHelp
)

// view is the contract each child view satisfies. View() renders the body
// region (chrome handles header/footer). Title/Status/Footer let chrome show
// per-view context.
type view interface {
	Init() tea.Cmd
	Update(tea.Msg) (view, tea.Cmd)
	View(width, bodyHeight int) string
	Title() string
	Status() string
	Footer() string
}

// App is the root Bubble Tea model.
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
		// children get window size via the same message
		v, cmd := a.scan.Update(msg)
		a.scan = v
		v, cmd2 := a.ai.Update(msg)
		a.ai = v
		v, cmd3 := a.help.Update(msg)
		a.help = v
		return a, tea.Batch(cmd, cmd2, cmd3)
	case tea.KeyMsg:
		// global keys first
		if key := msg.String(); !a.activeWantsKey(msg) {
			switch key {
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
			case "q":
				return a, tea.Quit
			}
		}
	}
	v, cmd := a.active().Update(msg)
	return a.setActive(v), cmd
}

// activeWantsKey lets the AI view consume keystrokes while its input is
// focused, so typed letters don't trigger global tab switches.
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

// View renders the full chrome.
func (a App) View() string {
	if a.width == 0 {
		return "initializing…"
	}
	bodyHeight := a.height - 4 // header + tabs + footer = 4 rows
	if bodyHeight < 5 {
		bodyHeight = 5
	}
	parts := []string{
		a.renderHeader(),
		a.renderTabs(),
		a.active().View(a.width, bodyHeight),
		a.renderFooter(),
	}
	return strings.Join(parts, "\n")
}

func (a App) renderHeader() string {
	left := titleStyle.Render(" LEAN-MAC ") + " " + headerStyle.Render(a.active().Title())
	used := a.diskTotal - a.diskFree
	pct := 0
	if a.diskTotal > 0 {
		pct = int(float64(used) / float64(a.diskTotal) * 100)
	}
	right := mutedStyle.Render(fmt.Sprintf("disk %s / %s (%d%%)  free %s",
		humanize.IBytes(uint64(used)),
		humanize.IBytes(uint64(a.diskTotal)),
		pct,
		humanize.IBytes(uint64(a.diskFree)),
	))
	if s := a.active().Status(); s != "" {
		right = mutedStyle.Render(s+"  ") + right
	}
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (a App) renderTabs() string {
	tabs := []struct {
		id    tabID
		label string
	}{
		{tabScan, "1·Scan"},
		{tabAI, "2·AI Cleanse"},
		{tabHelp, "3·Help"},
	}
	out := make([]string, 0, len(tabs))
	for _, t := range tabs {
		if t.id == a.tab {
			out = append(out, tabActive.Render(t.label))
		} else {
			out = append(out, tabInactive.Render(t.label))
		}
	}
	bar := strings.Join(out, "")
	padding := a.width - lipgloss.Width(bar)
	if padding < 0 {
		padding = 0
	}
	return bar + strings.Repeat(" ", padding)
}

func (a App) renderFooter() string {
	hint := a.active().Footer()
	suffix := "  ·  tab switch  ·  q quit"
	full := hint + suffix
	return statusBarStyle.Width(a.width).Render(full)
}
