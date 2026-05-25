package detectors

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/bagaspra16/lean-mac/internal/types"
)

// NodeModules finds node_modules directories under the user's home. Risk is
// SAFE for projects untouched for >90 days, MEDIUM otherwise (a reinstall
// brings them back, but it costs time and network).
type NodeModules struct{ Root string }

func (NodeModules) Name() string { return "node_modules" }

func (n NodeModules) Detect(ctx context.Context, emit func(types.Finding)) error {
	return filepath.WalkDir(n.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		// skip dotdirs & common heavy non-project trees
		if name == "Library" || name == ".Trash" || name == "Applications" {
			return filepath.SkipDir
		}
		if name == "node_modules" {
			risk := types.RiskMedium
			desc := "Node dependencies; reinstall with `npm/pnpm/yarn install`."
			if info, err := os.Stat(path); err == nil {
				if time.Since(info.ModTime()) > 90*24*time.Hour {
					risk = types.RiskSafe
					desc = "Stale (>90d) node dependencies."
				}
			}
			if f := finding(ctx, types.CatNode, path, risk, desc); f != nil {
				emit(*f)
			}
			return filepath.SkipDir // never recurse into node_modules
		}
		return nil
	})
}
