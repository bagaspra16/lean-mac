package monitor

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/dustin/go-humanize"
)

// Thresholds for disk pressure warnings.
type Thresholds struct {
	Warn     int64 // free bytes below which to warn
	Critical int64 // free bytes below which to escalate
}

// Default returns conservative thresholds: 25GB warn / 10GB critical.
func Default() Thresholds {
	return Thresholds{Warn: 25 * 1 << 30, Critical: 10 * 1 << 30}
}

// Run streams disk-pressure lines to w until ctx is cancelled.
func Run(ctx context.Context, w io.Writer, t Thresholds, interval time.Duration) error {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	emit := func() {
		free, total, err := fsutil.DiskUsage("/")
		if err != nil {
			fmt.Fprintf(w, "%s ERROR %v\n", time.Now().Format(time.RFC3339), err)
			return
		}
		level := "OK"
		if free < t.Critical {
			level = "CRITICAL"
		} else if free < t.Warn {
			level = "WARN"
		}
		fmt.Fprintf(w, "%s %-8s free=%s total=%s\n",
			time.Now().Format(time.RFC3339), level,
			humanize.IBytes(uint64(free)), humanize.IBytes(uint64(total)))
	}
	emit()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			emit()
		}
	}
}
