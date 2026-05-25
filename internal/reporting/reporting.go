package reporting

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bagaspra16/lean-mac/internal/types"
	"github.com/dustin/go-humanize"
)

func ScanJSON(w io.Writer, r *types.ScanReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func CleanJSON(w io.Writer, r *types.CleanReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// ScanText renders a human-readable scan summary.
func ScanText(w io.Writer, r *types.ScanReport) {
	totals := map[types.Category]int64{}
	counts := map[types.Category]int{}
	for _, f := range r.Findings {
		totals[f.Category] += f.Size
		counts[f.Category]++
	}
	fmt.Fprintf(w, "lean-mac scan — %s\n", r.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "duration: %dms  findings: %d  reclaimable: %s\n",
		r.DurationMS, len(r.Findings), humanize.IBytes(uint64(r.TotalBytes)))
	fmt.Fprintf(w, "disk: %s free / %s total\n\n",
		humanize.IBytes(uint64(r.DiskFree)), humanize.IBytes(uint64(r.DiskTotal)))
	fmt.Fprintf(w, "%-22s %12s  %5s\n", "category", "reclaimable", "items")
	fmt.Fprintln(w, strings.Repeat("-", 48))
	type row struct {
		cat   types.Category
		total int64
		count int
	}
	var rows []row
	for c, t := range totals {
		rows = append(rows, row{c, t, counts[c]})
	}
	// simple insertion sort by total desc
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].total > rows[j-1].total; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%-22s %12s  %5d\n", r.cat, humanize.IBytes(uint64(r.total)), r.count)
	}
}

// ScanMarkdown renders a portable scan report.
func ScanMarkdown(w io.Writer, r *types.ScanReport) {
	fmt.Fprintf(w, "# lean-mac scan report\n\n")
	fmt.Fprintf(w, "- started: %s\n", r.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "- duration: %dms\n", r.DurationMS)
	fmt.Fprintf(w, "- findings: %d\n", len(r.Findings))
	fmt.Fprintf(w, "- total reclaimable: **%s**\n", humanize.IBytes(uint64(r.TotalBytes)))
	fmt.Fprintf(w, "- disk free: %s / %s\n\n", humanize.IBytes(uint64(r.DiskFree)), humanize.IBytes(uint64(r.DiskTotal)))
	fmt.Fprintf(w, "## findings\n\n| risk | category | size | path |\n|---|---|---|---|\n")
	for _, f := range r.Findings {
		fmt.Fprintf(w, "| %s | %s | %s | `%s` |\n",
			f.Risk.String(), f.Category, humanize.IBytes(uint64(f.Size)), f.Path)
	}
}
