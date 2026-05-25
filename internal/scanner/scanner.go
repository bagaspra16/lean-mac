package scanner

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/bagaspra16/lean-mac/internal/types"
)

// Detector discovers Findings of one or more Categories. Detectors must be
// safe to run concurrently and must respect ctx cancellation.
type Detector interface {
	Name() string
	Detect(ctx context.Context, emit func(types.Finding)) error
}

// Scanner runs registered detectors concurrently and streams results.
type Scanner struct {
	detectors []Detector
}

func New(d ...Detector) *Scanner { return &Scanner{detectors: d} }

// Progress events stream back to the caller while scanning.
type Progress struct {
	Detector string
	Finding  *types.Finding
	Done     bool
	Err      error
}

// Run executes all detectors concurrently. progress is closed when finished.
func (s *Scanner) Run(ctx context.Context, progress chan<- Progress) *types.ScanReport {
	rpt := &types.ScanReport{StartedAt: time.Now(), HostPath: "/"}
	if free, total, err := fsutil.DiskUsage("/"); err == nil {
		rpt.DiskFree, rpt.DiskTotal = free, total
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, d := range s.detectors {
		wg.Add(1)
		go func(det Detector) {
			defer wg.Done()
			emit := func(f types.Finding) {
				f.RiskLabel = f.Risk.String()
				mu.Lock()
				rpt.Findings = append(rpt.Findings, f)
				rpt.TotalBytes += f.Size
				mu.Unlock()
				select {
				case progress <- Progress{Detector: det.Name(), Finding: &f}:
				case <-ctx.Done():
				}
			}
			err := det.Detect(ctx, emit)
			select {
			case progress <- Progress{Detector: det.Name(), Done: true, Err: err}:
			case <-ctx.Done():
			}
		}(d)
	}
	wg.Wait()
	close(progress)

	sort.Slice(rpt.Findings, func(i, j int) bool {
		return rpt.Findings[i].Size > rpt.Findings[j].Size
	})
	rpt.FinishedAt = time.Now()
	rpt.DurationMS = rpt.FinishedAt.Sub(rpt.StartedAt).Milliseconds()
	return rpt
}
