package detectors

import (
	"context"
	"os"

	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/bagaspra16/lean-mac/internal/types"
)

// finding constructs a Finding by sizing path on disk.
func finding(ctx context.Context, cat types.Category, path string, risk types.Risk, desc string) *types.Finding {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	size, _ := fsutil.DirSize(ctx, path)
	return &types.Finding{
		Category:    cat,
		Path:        path,
		Size:        size,
		LastMod:     info.ModTime(),
		Risk:        risk,
		Reversible:  false,
		Description: desc,
	}
}
