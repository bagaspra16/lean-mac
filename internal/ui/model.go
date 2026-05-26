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

// statusNotice is a transient one-line status shown below the legend.
type statusNotice struct {
	text    string
	expires time.Time
}

// scanView is the Scan tab. Compatible with the `view` interface.
type scanView struct {
	scn         *scanner.Scanner
	progressCh  chan scanner.Progress
	report      *types.ScanReport
	cleanReport *types.CleanReport
	findings    []types.Finding
	marked      map[string]bool
	cursor      int
	scroll      int // top row of visible window
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
	notice      *statusNotice
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
		return "Scanning disk"
	case phaseConfirm:
		return "Confirm cleanup"
	case phaseDone:
		return "Cleanup complete"
	default:
		return "Findings"
	}
}

func (v *scanView) Subtitle() string {
	switch v.phase {
	case phaseScanning:
		return "discovering reclaimable artifacts"
	case phaseConfirm:
		return "review then approve or cancel"
	case phaseDone:
		return "summary of what changed"
	default:
		if v.searching || v.search != "" {
			return fmt.Sprintf("searching for \"%s\"", v.search)
		}
		return "grouped by category · sorted by size"
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
	marked := v.countMarked()
	markedTag := ""
	if marked > 0 {
		var ms int64
		for _, f := range v.findings {
			if v.marked[f.Path] {
				ms += f.Size
			}
		}
		markedTag = fmt.Sprintf(" · %d marked (%s)", marked, humanize.IBytes(uint64(ms)))
	}
	return fmt.Sprintf("%d findings · %s reclaimable%s%s",
		len(v.findings), humanize.IBytes(uint64(total)), tag, markedTag)
}

func (v *scanView) Footer() []hint {
	if v.searching {
		return []hint{
			{"type", "filter text"},
			{"enter/esc", "exit search"},
			{"backspace", "delete char"},
		}
	}
	switch v.phase {
	case phaseConfirm:
		return []hint{{"y", "confirm delete"}, {"n/esc", "cancel"}}
	case phaseDone:
		return []hint{{"esc", "back to list"}}
	case phaseScanning:
		return []hint{{"scanning…", ""}}
	default:
		return []hint{
			{"j/k ↑↓", "navigate"},
			{"space", "mark/unmark"},
			{"a", "mark all SAFE"},
			{"A", "unmark all"},
			{"/", "search"},
			{"d", fmt.Sprintf("delete %d marked", v.countMarked())},
		}
	}
}

func (v *scanView) setNotice(text string) {
	v.notice = &statusNotice{text: text, expires: time.Now().Add(2 * time.Second)}
}

func (v *scanView) Update(msg tea.Msg) (view, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = msg.Width, msg.Height
		return v, nil
	case tickMsg:
		// expire notice
		if v.notice != nil && time.Now().After(v.notice.expires) {
			v.notice = nil
		}
		if v.phase == phaseScanning {
			return v, tickEvery()
		}
		if v.notice != nil {
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
		if v.search == "" {
			v.setNotice("Search cleared — showing all findings")
		}
	case tea.KeyBackspace:
		if len(v.search) > 0 {
			v.search = v.search[:len(v.search)-1]
		}
	case tea.KeyRunes:
		v.search += string(msg.Runes)
		v.cursor = 0
		v.scroll = 0
	}
	return v, nil
}

func (v *scanView) updateKey(msg tea.KeyMsg) (view, tea.Cmd) {
	visible := v.visibleFindings()
	maxRows := v.maxVisibleRows()
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(visible)-1 {
			v.cursor++
			// scroll window down if cursor moves past bottom
			if v.cursor >= v.scroll+maxRows {
				v.scroll = v.cursor - maxRows + 1
			}
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
			// scroll window up if cursor moves past top
			if v.cursor < v.scroll {
				v.scroll = v.cursor
			}
		}
	case "g":
		v.cursor = 0
		v.scroll = 0
		v.setNotice("Jumped to top")
	case "G":
		if len(visible) > 0 {
			v.cursor = len(visible) - 1
			v.scroll = max(0, v.cursor-maxRows+1)
			v.setNotice("Jumped to bottom")
		}
	case "ctrl+d":
		v.cursor = min(v.cursor+maxRows/2, len(visible)-1)
		v.scroll = max(0, v.cursor-maxRows+1)
	case "ctrl+u":
		v.cursor = max(0, v.cursor-maxRows/2)
		if v.cursor < v.scroll {
			v.scroll = v.cursor
		}
	case " ":
		if v.cursor < len(visible) {
			f := visible[v.cursor]
			if v.marked[f.Path] {
				v.marked[f.Path] = false
				v.setNotice(fmt.Sprintf("Unmarked: %s (%s)", shortPath(f.Path), humanize.IBytes(uint64(f.Size))))
			} else {
				v.marked[f.Path] = true
				v.setNotice(fmt.Sprintf("Marked for deletion: %s (%s)", shortPath(f.Path), humanize.IBytes(uint64(f.Size))))
			}
		}
	case "a":
		n := 0
		for _, f := range visible {
			if f.Risk == types.RiskSafe && !v.marked[f.Path] {
				v.marked[f.Path] = true
				n++
			}
		}
		if n > 0 {
			v.setNotice(fmt.Sprintf("Marked %d SAFE item(s) for deletion", n))
		} else {
			v.setNotice("All SAFE items already marked")
		}
	case "A":
		n := 0
		for k, ok := range v.marked {
			if ok {
				v.marked[k] = false
				n++
			}
		}
		if n > 0 {
			v.setNotice(fmt.Sprintf("Unmarked all — cleared %d item(s)", n))
		} else {
			v.setNotice("Nothing was marked")
		}
	case "/":
		if v.phase == phaseReady {
			v.searching = true
			v.search = ""
			v.cursor = 0
			v.scroll = 0
		}
	case "d":
		if v.phase == phaseReady && v.countMarked() > 0 {
			v.phase = phaseConfirm
		} else if v.phase == phaseReady {
			v.setNotice("Nothing marked — press space to mark items, then d to delete")
		}
	case "y":
		if v.phase == phaseConfirm {
			return v, v.runClean()
		}
	case "n", "esc":
		if v.phase == phaseConfirm {
			v.phase = phaseReady
			v.setNotice("Delete cancelled — items remain marked")
		} else if v.phase == phaseDone {
			v.phase = phaseReady
			v.cleanReport = nil
			v.marked = map[string]bool{}
			v.setNotice("Back to results — marks cleared")
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
		return cleanDoneMsg{report: c.Clean(context.Background(), findings, nil)}
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

// maxVisibleRows returns the number of body rows available for the findings table.
func (v *scanView) maxVisibleRows() int {
	// legend(1) + notice(1) + breathing room(1)
	r := v.height - 3
	if r < 5 {
		r = 5
	}
	return r
}

func (v *scanView) View(width, height int) string {
	v.width, v.height = width, height
	body := ""
	switch v.phase {
	case phaseDone:
		body = v.viewDone()
	case phaseConfirm:
		body = v.viewTable() + "\n\n" + v.viewConfirm()
	default:
		body = v.legendRow() + "\n" + v.noticeRow() + v.viewTable()
	}
	return body
}

// legendRow renders an inline risk legend + search banner.
func (v *scanView) legendRow() string {
	parts := []string{
		dimStyle.Render("risk:"),
		riskChip("SAFE") + " " + dimStyle.Render("auto-regen"),
		riskChip("MEDIUM") + " " + dimStyle.Render("rebuild needed"),
		riskChip("DANGEROUS") + " " + dimStyle.Render("may lose data"),
	}
	legend := strings.Join(parts, "  ")

	if v.search != "" {
		searchInfo := "  " + searchBannerStyle.Render(" SEARCH ") + " " +
			dimStyle.Render("filtering by: ") + searchTermStyle.Render("\""+v.search+"\"")
		if v.searching {
			searchInfo += dimStyle.Render(" (typing…)")
		} else {
			searchInfo += dimStyle.Render(" — press / to edit, esc to clear")
		}
		return legend + searchInfo
	}
	if v.searching {
		return legend + "  " + searchBannerStyle.Render(" SEARCH ") +
			" " + dimStyle.Render("type to filter…")
	}
	return legend
}

// noticeRow returns a transient notice line (or empty string).
func (v *scanView) noticeRow() string {
	if v.notice == nil {
		return ""
	}
	return "\n" + lipgloss.NewStyle().Foreground(colorAccent2).Italic(true).Render("  → "+v.notice.text)
}

func (v *scanView) viewTable() string {
	display := v.visibleFindings()
	if len(display) == 0 {
		if v.phase == phaseScanning {
			return v.viewScanningEmpty()
		}
		if v.search != "" {
			return "\n" + mutedStyle.Render(fmt.Sprintf(
				"  No results for \"%s\" — try a shorter term or press esc to clear", v.search))
		}
		return mutedStyle.Render("  no findings")
	}

	maxRows := v.maxVisibleRows()
	catTotals := map[types.Category]int64{}
	catCounts := map[types.Category]int{}
	for _, f := range display {
		catTotals[f.Category] += f.Size
		catCounts[f.Category]++
	}

	// Clamp scroll
	if v.scroll < 0 {
		v.scroll = 0
	}
	if v.scroll > len(display)-1 {
		v.scroll = len(display) - 1
	}

	var rows []string
	var lastCat types.Category
	rowCount := 0
	for i := v.scroll; i < len(display) && rowCount < maxRows; i++ {
		f := display[i]
		if f.Category != lastCat {
			rows = append(rows, v.renderCategoryHeader(f.Category, catTotals[f.Category], catCounts[f.Category]))
			rowCount++
			lastCat = f.Category
		}
		rows = append(rows, v.renderRow(f, i == v.cursor))
		rowCount++
	}

	// Scroll indicator
	total := len(display)
	if total > maxRows {
		pct := 0
		if total > 1 {
			pct = v.cursor * 100 / (total - 1)
		}
		indicator := scrollStyle.Render(fmt.Sprintf(
			"  ↕ %d/%d  (%d%%)  g=top  G=bottom  ctrl+d/u=page",
			v.cursor+1, total, pct))
		rows = append(rows, indicator)
	}

	return "\n" + strings.Join(rows, "\n")
}

func (v *scanView) renderCategoryHeader(cat types.Category, total int64, count int) string {
	head := fmt.Sprintf("▾ %s  %s  %s",
		lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(string(cat)),
		mutedStyle.Render(humanize.IBytes(uint64(total))),
		subtleStyle.Render(fmt.Sprintf("(%d items)", count)),
	)
	if blurb := categoryBlurb(cat); blurb != "" {
		head += "\n  " + dimStyle.Render(blurb)
	}
	return head
}

func (v *scanView) renderRow(f types.Finding, selected bool) string {
	marker := "  "
	if v.marked[f.Path] {
		marker = markedStyle.Render("● ")
	}
	chip := riskChip(f.Risk.String())
	size := humanize.IBytes(uint64(f.Size))
	path := f.Path
	maxPath := v.width - 32
	if maxPath > 0 && len(path) > maxPath {
		path = "…" + path[len(path)-maxPath+1:]
	}
	line := fmt.Sprintf("   %s%s %10s  %s", marker, chip, size, path)
	if selected {
		return selectedRowStyle.Render(line)
	}
	return rowStyle.Render(line)
}

func (v *scanView) viewScanningEmpty() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[(time.Now().UnixMilli()/100)%int64(len(frames))]
	dot := lipgloss.NewStyle().Foreground(colorAccent).Render(frame)
	elapsed := time.Since(v.startedAt).Round(time.Second)
	lines := []string{
		"",
		"  " + dot + "  " + headerStyle.Render("Scanning your home directory and known cache paths"),
		"  " + dimStyle.Render(fmt.Sprintf("Elapsed: %s · %d findings so far · active detector: %s",
			elapsed, v.scanned, v.activeName)),
		"",
		"  " + dimStyle.Render("This typically takes 10-30 seconds. Detectors run concurrently;"),
		"  " + dimStyle.Render("findings stream in as they are discovered."),
	}
	return strings.Join(lines, "\n")
}

func (v *scanView) viewConfirm() string {
	marked := v.markedFindings()
	var total int64
	risk := types.RiskSafe
	for _, f := range marked {
		total += f.Size
		if f.Risk > risk {
			risk = f.Risk
		}
	}
	heading := riskChip(risk.String()) + "  " +
		lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("Confirm Deletion")

	mode := "LIVE — files will be permanently deleted"
	if v.dryRun {
		mode = "DRY-RUN — nothing will actually be deleted"
	}

	// Show first 3 marked paths
	var previewLines []string
	for i, f := range marked {
		if i >= 3 {
			previewLines = append(previewLines, dimStyle.Render(fmt.Sprintf("  … and %d more item(s)", len(marked)-3)))
			break
		}
		chip := riskChip(f.Risk.String())
		previewLines = append(previewLines, fmt.Sprintf("  %s  %s  %s",
			chip,
			mutedStyle.Render(fmt.Sprintf("%10s", humanize.IBytes(uint64(f.Size)))),
			dimStyle.Render(shortPath(f.Path))))
	}

	lines := []string{
		heading,
		"",
		fmt.Sprintf("%d item(s) selected   Reclaimable: %s",
			len(marked),
			lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(humanize.IBytes(uint64(total)))),
		"",
	}
	lines = append(lines, previewLines...)
	lines = append(lines,
		"",
		dimStyle.Render("Mode: "+mode),
		"",
		approveKeyStyle.Render(" y ")+" confirm    "+rejectKeyStyle.Render(" n/esc ")+" cancel",
	)
	return approvalBoxForRisk(risk.String()).Render(strings.Join(lines, "\n"))
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
	fmt.Fprintf(&b, "%s  mode: %s\n",
		doneBadgeStyle.Render(" DONE "), mode)
	fmt.Fprintf(&b, "%s  reclaimed: %s\n",
		doneBadgeStyle.Render("      "),
		lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(humanize.IBytes(uint64(v.cleanReport.BytesFreed))))
	if v.cleanReport.DiskBefore > 0 {
		delta := v.cleanReport.DiskAfter - v.cleanReport.DiskBefore
		fmt.Fprintf(&b, "%s  disk free Δ: %s\n",
			doneBadgeStyle.Render("      "),
			humanize.IBytes(uint64(delta)))
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
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Press esc to return to the findings list."))
	return b.String()
}

// shortPath truncates a path to the last 2 components for display.
func shortPath(p string) string {
	parts := strings.Split(p, "/")
	if len(parts) <= 3 {
		return p
	}
	return "…/" + strings.Join(parts[len(parts)-2:], "/")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
