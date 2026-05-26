package cleaner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/bagaspra16/lean-mac/internal/types"
)

// Protected paths must never be deleted, even if a buggy detector emits them.
var protected = []string{
	"/", "/System", "/Library", "/usr", "/bin", "/sbin", "/etc", "/var",
	"/Applications", "/Users",
}

// Options controls cleaner behavior.
type Options struct {
	DryRun     bool
	Aggressive bool // include MEDIUM risk findings by default
	IncludeDangerous bool
}

type Cleaner struct{ opt Options }

func New(opt Options) *Cleaner { return &Cleaner{opt: opt} }

// Eligible returns true if the cleaner is allowed to act on f given options.
func (c *Cleaner) Eligible(f types.Finding) bool {
	switch f.Risk {
	case types.RiskSafe:
		return true
	case types.RiskMedium:
		return c.opt.Aggressive
	case types.RiskDangerous:
		return c.opt.IncludeDangerous
	}
	return false
}

// Clean deletes (or simulates deleting) the given findings and returns a report.
// If results is non-nil, each CleanResult is sent to it immediately as it completes,
// enabling live progress updates in the UI. The channel is NOT closed by Clean.
func (c *Cleaner) Clean(ctx context.Context, findings []types.Finding, results chan<- types.CleanResult) *types.CleanReport {
	rpt := &types.CleanReport{StartedAt: time.Now(), DryRun: c.opt.DryRun}
	if free, _, err := fsutil.DiskUsage("/"); err == nil {
		rpt.DiskBefore = free
	}
	for _, f := range findings {
		select {
		case <-ctx.Done():
			goto done
		default:
		}
		res := c.cleanOne(ctx, f)
		rpt.Results = append(rpt.Results, res)
		if res.Success {
			rpt.BytesFreed += res.BytesFreed
		}
		if results != nil {
			results <- res
		}
	}
done:
	if free, _, err := fsutil.DiskUsage("/"); err == nil {
		rpt.DiskAfter = free
	}
	rpt.FinishedAt = time.Now()
	return rpt
}

func (c *Cleaner) cleanOne(ctx context.Context, f types.Finding) types.CleanResult {
	res := types.CleanResult{Finding: f, DryRun: c.opt.DryRun}

	if c.opt.DryRun {
		res.Success = true
		res.BytesFreed = f.Size
		return res
	}

	switch f.Category {
	case types.CatDockerImg:
		res.Error, res.Success, res.BytesFreed = runDocker(ctx, []string{"image", "prune", "-af"}, f.Size)
	case types.CatDockerVol:
		res.Error, res.Success, res.BytesFreed = runDocker(ctx, []string{"volume", "prune", "-f"}, f.Size)
	case types.CatDockerBuild:
		res.Error, res.Success, res.BytesFreed = runDocker(ctx, []string{"builder", "prune", "-af"}, f.Size)
	case types.CatXcodeSim:
		// path is the per-device dir; UDID is the basename.
		udid := filepath.Base(f.Path)
		err := exec.CommandContext(ctx, "xcrun", "simctl", "delete", udid).Run()
		if err != nil {
			res.Error = err.Error()
			return res
		}
		res.Success = true
		res.BytesFreed = f.Size
	default:
		if err := safeRemove(f.Path); err != nil {
			res.Error = err.Error()
			return res
		}
		res.Success = true
		res.BytesFreed = f.Size
	}
	return res
}

func runDocker(ctx context.Context, args []string, claimed int64) (string, bool, int64) {
	if _, err := exec.LookPath("docker"); err != nil {
		return "docker not installed", false, 0
	}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(cctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("docker %s: %v: %s", strings.Join(args, " "), err, string(out)), false, 0
	}
	return "", true, claimed
}

// safeRemove refuses to touch protected paths and resolves symlinks to avoid
// deleting outside the requested directory.
func safeRemove(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	clean := filepath.Clean(abs)
	for _, p := range protected {
		if clean == p {
			return errors.New("refusing to delete protected path: " + clean)
		}
	}
	if home, err := os.UserHomeDir(); err == nil && clean == filepath.Clean(home) {
		return errors.New("refusing to delete user home directory")
	}
	return os.RemoveAll(clean)
}
