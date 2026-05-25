package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bagaspra16/lean-mac/internal/cleaner"
	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/bagaspra16/lean-mac/internal/scanner"
	"github.com/bagaspra16/lean-mac/internal/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type phase int

const (
	phaseScanning phase = iota
	phaseReady
	phaseConfirm
	phaseCleaning
	phaseDone
)

type scanProgressMsg struct{ p scanner.Progress }
type scanDoneMsg struct{ report *types.ScanReport }
type cleanDoneMsg struct{ report *types.CleanReport }
type tickMsg time.Time

type Model struct {
	scn         *scanner.Scanner
	progressCh  chan scanner.Progress
	report      *types.ScanReport
	cleanReport *types.CleanReport
	findings    []types.Finding
	marked      map[string]bool
	cursor      int
	width       int
	height      int
	phase       phase
	search      string
	searching   bool
	activeName  string
	scanned     int
	startedAt   time.Time
	diskFree    int64
	diskTotal   int64
	dryRun      bool
	scanDone    chan *types.ScanReport
}

func NewModel(scn *scanner.Scanner, dryRun bool) Model {
	free, total, _ := fsutil.DiskUsage("/")
	return Model{
		scn:        scn,
		marked:     map[string]bool{},
		phase:      phaseScanning,
		startedAt:  time.Now(),
		diskFree:   free,
		diskTotal:  total,
		dryRun:     dryRun,
		progressCh: make(chan scanner.Progress, 64),
		scanDone:   make(chan *types.ScanReport, 1),
	}
}

func (m Model) Init() tea.Cmd {
	go func() {
		m.scanDone <- m.scn.Run(context.Background(), m.progressCh)
	}()
	return tea.Batch(waitProgress(m.progressCh, m.scanDone), tickEvery())
}

func waitProgress(ch chan scanner.Progress, done chan *types.ScanReport) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-ch
		if !ok {
			return scanDoneMsg{report: <-done}
		}
		return scanProgressMsg{p: p}
	}
}

func tickEvery() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if m.phase == phaseScanning {
			return m, tickEvery()
		}
		return m, nil

	case scanProgressMsg:
		if msg.p.Finding != nil {
			m.findings = append(m.findings, *msg.p.Finding)
			m.scanned++
		}
		m.activeName = msg.p.Detector
		return m, waitProgress(m.progressCh, m.scanDone)

	case scanDoneMsg:
		m.report = msg.report
		m.findings = msg.report.Findings
		m.phase = phaseReady
		return m, nil

	case cleanDoneMsg:
		m.cleanReport = msg.report
		m.phase = phaseDone
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			return m.updateSearch(msg)
		}
		return m.updateKey(msg)
	}
	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEsc:
		m.searching = false
		return m, nil
	case tea.KeyBackspace:
		if len(m.search) > 0 {
			m.search = m.search[:len(m.search)-1]
		}
		return m, nil
	case tea.KeyRunes:
		m.search += string(msg.Runes)
		return m, nil
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := m.visibleFindings()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(visible)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(visible) > 0 {
			m.cursor = len(visible) - 1
		}
	case " ":
		if m.cursor < len(visible) {
			f := visible[m.cursor]
			m.marked[f.Path] = !m.marked[f.Path]
		}
	case "a":
		for _, f := range visible {
			if f.Risk == types.RiskSafe {
				m.marked[f.Path] = true
			}
		}
	case "/":
		if m.phase == phaseReady {
			m.searching = true
			m.search = ""
		}
	case "d":
		if m.phase == phaseReady && m.countMarked() > 0 {
			m.phase = phaseConfirm
		}
	case "y":
		if m.phase == phaseConfirm {
			return m, m.runClean()
		}
	case "n", "esc":
		if m.phase == phaseConfirm {
			m.phase = phaseReady
		}
	}
	return m, nil
}

func (m Model) countMarked() int {
	n := 0
	for _, v := range m.marked {
		if v {
			n++
		}
	}
	return n
}

func (m Model) markedFindings() []types.Finding {
	var out []types.Finding
	for _, f := range m.findings {
		if m.marked[f.Path] {
			out = append(out, f)
		}
	}
	return out
}

func (m Model) runClean() tea.Cmd {
	findings := m.markedFindings()
	dry := m.dryRun
	return func() tea.Msg {
		c := cleaner.New(cleaner.Options{DryRun: dry, Aggressive: true, IncludeDangerous: true})
		return cleanDoneMsg{report: c.Clean(context.Background(), findings)}
	}
}

// visibleFindings returns the findings to render, grouped by category and
// sorted by group total then by per-item size. Cursor indexes into this slice
// directly, so display order == selection order.
func (m Model) visibleFindings() []types.Finding {
	src := m.findings
	if m.search != "" {
		q := strings.ToLower(m.search)
		src = src[:0:0]
		for _, f := range m.findings {
			if strings.Contains(strings.ToLower(string(f.Category)), q) ||
				strings.Contains(strings.ToLower(f.Path), q) {
				src = append(src, f)
			}
		}
	}
	type group struct {
		cat   types.Category
		items []types.Finding
		total int64
	}
	groups := map[types.Category]*group{}
	for _, f := range src {
		g, ok := groups[f.Category]
		if !ok {
			g = &group{cat: f.Category}
			groups[f.Category] = g
		}
		g.items = append(g.items, f)
		g.total += f.Size
	}
	var sorted []*group
	for _, g := range groups {
		sort.Slice(g.items, func(i, j int) bool { return g.items[i].Size > g.items[j].Size })
		sorted = append(sorted, g)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].total > sorted[j].total })
	out := make([]types.Finding, 0, len(src))
	for _, g := range sorted {
		out = append(out, g.items...)
	}
	return out
}

// --- view ---

func (m Model) View() string {
	if m.width == 0 {
		return "initializing…"
	}
	switch m.phase {
	case phaseDone:
		return m.viewDone()
	case phaseConfirm:
		return m.viewMain() + "\n" + m.viewConfirm()
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	var b strings.Builder
	b.WriteString(m.viewHeader())
	b.WriteString("\n")
	if m.phase == phaseScanning {
		b.WriteString(m.viewScanProgress())
		b.WriteString("\n")
	}
	b.WriteString(m.viewTable())
	b.WriteString("\n")
	b.WriteString(m.viewStatus())
	return b.String()
}

func (m Model) viewHeader() string {
	title := titleStyle.Render(" LEAN-MAC ")
	used := m.diskTotal - m.diskFree
	pct := 0
	if m.diskTotal > 0 {
		pct = int(float64(used) / float64(m.diskTotal) * 100)
	}
	disk := fmt.Sprintf("disk %s / %s (%d%% used, %s free)",
		humanize.IBytes(uint64(used)),
		humanize.IBytes(uint64(m.diskTotal)),
		pct,
		humanize.IBytes(uint64(m.diskFree)),
	)
	total := int64(0)
	for _, f := range m.findings {
		total += f.Size
	}
	reclaim := fmt.Sprintf("reclaimable %s", humanize.IBytes(uint64(total)))
	if m.dryRun {
		reclaim += "  " + mutedStyle.Render("[dry-run]")
	}
	right := mutedStyle.Render(disk) + "  " + headerStyle.Render(reclaim)
	left := title
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) viewScanProgress() string {
	dur := time.Since(m.startedAt).Truncate(time.Second)
	return mutedStyle.Render(fmt.Sprintf("scanning… %s | active: %s | findings: %d",
		dur, m.activeName, m.scanned))
}

func (m Model) viewTable() string {
	display := m.visibleFindings()
	if len(display) == 0 {
		if m.phase == phaseScanning {
			return mutedStyle.Render("  (scanning…)")
		}
		return mutedStyle.Render("  (no findings)")
	}
	maxRows := m.height - 8
	if maxRows < 5 {
		maxRows = 5
	}
	// emit category section headers when the category changes; the cursor
	// indexes only the item rows (not headers), which matches visibleFindings.
	var rows []string
	var lastCat types.Category
	var catTotal int64
	var catCount int
	// pre-compute per-category totals from display
	catTotals := map[types.Category]int64{}
	catCounts := map[types.Category]int{}
	for _, f := range display {
		catTotals[f.Category] += f.Size
		catCounts[f.Category]++
	}
	// window around cursor so a long list scrolls
	start := 0
	if m.cursor > maxRows-3 {
		start = m.cursor - (maxRows - 3)
	}
	for i := start; i < len(display); i++ {
		f := display[i]
		if f.Category != lastCat {
			catTotal = catTotals[f.Category]
			catCount = catCounts[f.Category]
			rows = append(rows, fmt.Sprintf("▾ %s  %s  (%d items)",
				headerStyle.Render(string(f.Category)),
				mutedStyle.Render(humanize.IBytes(uint64(catTotal))),
				catCount,
			))
			lastCat = f.Category
		}
		rows = append(rows, m.renderRow(f, i == m.cursor))
		if len(rows) >= maxRows {
			break
		}
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderRow(f types.Finding, selected bool) string {
	marker := " "
	if m.marked[f.Path] {
		marker = markedStyle.Render("●")
	}
	var risk string
	switch f.Risk {
	case types.RiskSafe:
		risk = riskSafe.Render("SAFE  ")
	case types.RiskMedium:
		risk = riskMed.Render("MED   ")
	case types.RiskDangerous:
		risk = riskDang.Render("DANGER")
	}
	size := humanize.IBytes(uint64(f.Size))
	path := f.Path
	maxPath := m.width - 30
	if maxPath > 0 && len(path) > maxPath {
		path = "…" + path[len(path)-maxPath+1:]
	}
	line := fmt.Sprintf(" %s %s %10s  %s", marker, risk, size, path)
	if selected {
		return selectedRowStyle.Render(line)
	}
	return rowStyle.Render(line)
}

func (m Model) viewStatus() string {
	var bar string
	if m.searching {
		bar = "/" + m.search + "█  (esc to cancel)"
	} else {
		marked := m.countMarked()
		bar = fmt.Sprintf("j/k move • space mark • a mark-safe • / search • d delete (%d marked) • q quit", marked)
	}
	return statusBarStyle.Width(m.width).Render(bar)
}

func (m Model) viewConfirm() string {
	marked := m.markedFindings()
	var total int64
	for _, f := range marked {
		total += f.Size
	}
	mode := "DELETE"
	if m.dryRun {
		mode = "DRY-RUN (no files will be deleted)"
	}
	body := fmt.Sprintf("%s\n\n%d items, %s reclaimable\n\n[y] confirm  [n] cancel",
		mode, len(marked), humanize.IBytes(uint64(total)))
	return modalStyle.Render(body)
}

func (m Model) viewDone() string {
	if m.cleanReport == nil {
		return "no cleanup ran."
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(" CLEANUP COMPLETE "))
	b.WriteString("\n\n")
	mode := "executed"
	if m.cleanReport.DryRun {
		mode = "dry-run"
	}
	b.WriteString(fmt.Sprintf("mode: %s\n", mode))
	b.WriteString(fmt.Sprintf("reclaimed: %s\n", humanize.IBytes(uint64(m.cleanReport.BytesFreed))))
	if m.cleanReport.DiskBefore > 0 {
		delta := m.cleanReport.DiskAfter - m.cleanReport.DiskBefore
		b.WriteString(fmt.Sprintf("disk free delta: %s\n", humanize.IBytes(uint64(delta))))
	}
	b.WriteString("\n")
	for _, r := range m.cleanReport.Results {
		status := riskSafe.Render("✓")
		if !r.Success {
			status = riskDang.Render("✗")
		}
		b.WriteString(fmt.Sprintf(" %s %-20s %10s  %s",
			status, r.Finding.Category, humanize.IBytes(uint64(r.BytesFreed)), r.Finding.Path))
		if r.Error != "" {
			b.WriteString("  " + mutedStyle.Render(r.Error))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + mutedStyle.Render("press q to exit"))
	return b.String()
}
