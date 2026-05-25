package detectors

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bagaspra16/lean-mac/internal/types"
)

// RustTarget finds `target/` directories that are siblings of Cargo.toml.
type RustTarget struct{ Root string }

func (RustTarget) Name() string { return "rust-target" }

func (r RustTarget) Detect(ctx context.Context, emit func(types.Finding)) error {
	return filepath.WalkDir(r.Root, func(path string, d os.DirEntry, err error) error {
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
		if name == "Library" || name == ".Trash" || name == "node_modules" || name == "target" {
			if name == "target" {
				parent := filepath.Dir(path)
				if _, err := os.Stat(filepath.Join(parent, "Cargo.toml")); err == nil {
					if f := finding(ctx, types.CatRustTarget, path,
						types.RiskSafe, "Cargo build output; recompiles on next build."); f != nil {
						emit(*f)
					}
				}
			}
			return filepath.SkipDir
		}
		return nil
	})
}
