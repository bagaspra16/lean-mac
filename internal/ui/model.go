package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bagaspra16/lean-mac/internal/cleaner"
	"github.com/bagaspra16/lean-mac/internal/detectors"
	"github.com/bagaspra16/lean-mac/internal/scanner"
	"github.com/bagaspra16/lean-mac/internal/types"

	tea "github.com/charmbracelet/bubbletea"
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

// scanView is the Scan tab. Compatible with the `view` interface.
type scanView struct {
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
	dryRun      bool
	scanDone    chan *types.ScanReport
}

func newScanView(dryRun bool) *scanView {
	s := scanner.New(detectors.Default()...)
	return &scanView{
		scn:        s,
		marked:     map[string]bool{},
		phase:      phaseScanning,
		startedAt:  time.Now(),
		dryRun:     dryRun,
		progressCh: make(chan scanner.Progress, 64),
		scanDone:   make(chan *types.ScanReport, 1),
	}
}

func (v *scanView) Init() tea.Cmd {
	go func() {
		v.scanDone <- v.scn.Run(context.Background(), v.progressCh)
	}()
	return tea.Batch(waitProgress(v.progressCh, v.scanDone), tickEvery())
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

func (v *scanView) Title() string {
	switch v.phase {
	case phaseScanning:
		return "Scanning"
	case phaseConfirm:
		return "Confirm cleanup"
	case phaseDone:
		return "Cleanup complete"
	default:
		return "Findings"
	}
}

func (v *scanView) Status() string {
	if v.phase == phaseScanning {
		return fmt.Sprintf("active: %s · %d found", v.activeName, v.scanned)
	}
	var total int64
	for _, f := range v.findings {
		total += f.Size
	}
	tag := ""
	if v.dryRun {
		tag = " · dry-run"
	}
	return fmt.Sprintf("reclaimable %s%s", humanize.IBytes(uint64(total)), tag)
}

func (v *scanView) Footer() string {
	if v.searching {
		return "/" + v.search + " · enter/esc done"
	}
	switch v.phase {
	case phaseConfirm:
		return "y confirm · n cancel"
	case phaseDone:
		return "esc back to findings"
	default:
		return fmt.Sprintf("j/k move · space mark · a mark-safe · / search · d delete (%d marked)", v.countMarked())
	}
}

func (v *scanView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = msg.Width, msg.Height
		return v, nil
	case tickMsg:
		if v.phase == phaseScanning {
			return v, tickEvery()
		}
		return v, nil
	case scanProgressMsg:
		if msg.p.Finding != nil {
			v.findings = append(v.findings, *msg.p.Finding)
			v.scanned++
		}
		v.activeName = msg.p.Detector
		return v, waitProgress(v.progressCh, v.scanDone)
	case scanDoneMsg:
		v.report = msg.report
		v.findings = msg.report.Findings
		v.phase = phaseReady
		return v, nil
	case cleanDoneMsg:
		v.cleanReport = msg.report
		v.phase = phaseDone
		return v, nil
	case tea.KeyMsg:
		if v.searching {
			return v.updateSearch(msg)
		}
		return v.updateKey(msg)
	}
	return v, nil
}

func (v *scanView) updateSearch(msg tea.KeyMsg) (view, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEsc:
		v.searching = false
	case tea.KeyBackspace:
		if len(v.search) > 0 {
			v.search = v.search[:len(v.search)-1]
		}
	case tea.KeyRunes:
		v.search += string(msg.Runes)
	}
	return v, nil
}

func (v *scanView) updateKey(msg tea.KeyMsg) (view, tea.Cmd) {
	visible := v.visibleFindings()
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(visible)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		}
	case "g":
		v.cursor = 0
	case "G":
		if len(visible) > 0 {
			v.cursor = len(visible) - 1
		}
	case " ":
		if v.cursor < len(visible) {
			f := visible[v.cursor]
			v.marked[f.Path] = !v.marked[f.Path]
		}
	case "a":
		for _, f := range visible {
			if f.Risk == types.RiskSafe {
				v.marked[f.Path] = true
			}
		}
	case "/":
		if v.phase == phaseReady {
			v.searching = true
			v.search = ""
		}
	case "d":
		if v.phase == phaseReady && v.countMarked() > 0 {
			v.phase = phaseConfirm
		}
	case "y":
		if v.phase == phaseConfirm {
			return v, v.runClean()
		}
	case "n", "esc":
		if v.phase == phaseConfirm {
			v.phase = phaseReady
		} else if v.phase == phaseDone {
			v.phase = phaseReady
			v.cleanReport = nil
			v.marked = map[string]bool{}
		}
	}
	return v, nil
}

func (v *scanView) countMarked() int {
	n := 0
	for _, ok := range v.marked {
		if ok {
			n++
		}
	}
	return n
}

func (v *scanView) markedFindings() []types.Finding {
	var out []types.Finding
	for _, f := range v.findings {
		if v.marked[f.Path] {
			out = append(out, f)
		}
	}
	return out
}

func (v *scanView) runClean() tea.Cmd {
	findings := v.markedFindings()
	dry := v.dryRun
	return func() tea.Msg {
		c := cleaner.New(cleaner.Options{DryRun: dry, Aggressive: true, IncludeDangerous: true})
		return cleanDoneMsg{report: c.Clean(context.Background(), findings)}
	}
}

func (v *scanView) visibleFindings() []types.Finding {
	src := v.findings
	if v.search != "" {
		q := strings.ToLower(v.search)
		filtered := make([]types.Finding, 0, len(src))
		for _, f := range src {
			if strings.Contains(strings.ToLower(string(f.Category)), q) ||
				strings.Contains(strings.ToLower(f.Path), q) {
				filtered = append(filtered, f)
			}
		}
		src = filtered
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

func (v *scanView) View(width, height int) string {
	v.width, v.height = width, height+4 // map back to internal height for sizing helpers
	switch v.phase {
	case phaseDone:
		return v.viewDone()
	case phaseConfirm:
		return v.viewTable() + "\n\n" + v.viewConfirm()
	default:
		return v.viewTable()
	}
}

func (v *scanView) viewTable() string {
	display := v.visibleFindings()
	if len(display) == 0 {
		if v.phase == phaseScanning {
			return mutedStyle.Render("  scanning…")
		}
		return mutedStyle.Render("  no findings")
	}
	maxRows := v.height - 8
	if maxRows < 5 {
		maxRows = 5
	}
	catTotals := map[types.Category]int64{}
	catCounts := map[types.Category]int{}
	for _, f := range display {
		catTotals[f.Category] += f.Size
		catCounts[f.Category]++
	}
	start := 0
	if v.cursor > maxRows-3 {
		start = v.cursor - (maxRows - 3)
	}
	var rows []string
	var lastCat types.Category
	for i := start; i < len(display); i++ {
		f := display[i]
		if f.Category != lastCat {
			rows = append(rows, fmt.Sprintf("▾ %s  %s  (%d items)",
				headerStyle.Render(string(f.Category)),
				mutedStyle.Render(humanize.IBytes(uint64(catTotals[f.Category]))),
				catCounts[f.Category],
			))
			lastCat = f.Category
		}
		rows = append(rows, v.renderRow(f, i == v.cursor))
		if len(rows) >= maxRows {
			break
		}
	}
	return strings.Join(rows, "\n")
}

func (v *scanView) renderRow(f types.Finding, selected bool) string {
	marker := " "
	if v.marked[f.Path] {
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
	maxPath := v.width - 30
	if maxPath > 0 && len(path) > maxPath {
		path = "…" + path[len(path)-maxPath+1:]
	}
	line := fmt.Sprintf(" %s %s %10s  %s", marker, risk, size, path)
	if selected {
		return selectedRowStyle.Render(line)
	}
	return rowStyle.Render(line)
}

func (v *scanView) viewConfirm() string {
	marked := v.markedFindings()
	var total int64
	for _, f := range marked {
		total += f.Size
	}
	mode := "DELETE"
	if v.dryRun {
		mode = "DRY-RUN (nothing will be deleted)"
	}
	body := fmt.Sprintf("%s\n\n%d items, %s reclaimable\n\n[y] confirm   [n] cancel",
		mode, len(marked), humanize.IBytes(uint64(total)))
	return modalStyle.Render(body)
}

func (v *scanView) viewDone() string {
	if v.cleanReport == nil {
		return "no cleanup ran."
	}
	var b strings.Builder
	mode := "executed"
	if v.cleanReport.DryRun {
		mode = "dry-run"
	}
	fmt.Fprintf(&b, "mode: %s\n", mode)
	fmt.Fprintf(&b, "reclaimed: %s\n", humanize.IBytes(uint64(v.cleanReport.BytesFreed)))
	if v.cleanReport.DiskBefore > 0 {
		delta := v.cleanReport.DiskAfter - v.cleanReport.DiskBefore
		fmt.Fprintf(&b, "disk free delta: %s\n", humanize.IBytes(uint64(delta)))
	}
	b.WriteString("\n")
	for _, r := range v.cleanReport.Results {
		status := riskSafe.Render("✓")
		if !r.Success {
			status = riskDang.Render("✗")
		}
		fmt.Fprintf(&b, " %s %-22s %10s  %s",
			status, r.Finding.Category, humanize.IBytes(uint64(r.BytesFreed)), r.Finding.Path)
		if r.Error != "" {
			fmt.Fprintf(&b, "  %s", mutedStyle.Render(r.Error))
		}
		b.WriteString("\n")
	}
	return b.String()
}
