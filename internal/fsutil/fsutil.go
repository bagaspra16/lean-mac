package fsutil

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
)

// DirSize walks dir and sums file sizes. It is cancellable via ctx and safe
// against permission errors (skipped silently). Returns bytes and number of files.
func DirSize(ctx context.Context, dir string) (int64, int64) {
	var size int64
	var count int64
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		atomic.AddInt64(&size, info.Size())
		atomic.AddInt64(&count, 1)
		return nil
	})
	return size, count
}

// DiskUsage returns free and total bytes on the volume containing path.
func DiskUsage(path string) (free, total int64, err error) {
	var stat syscall.Statfs_t
	if err = syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	free = int64(stat.Bavail) * int64(stat.Bsize)
	total = int64(stat.Blocks) * int64(stat.Bsize)
	return
}

// Exists returns true if path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ExpandHome resolves a leading ~ in a path.
func ExpandHome(p string) string {
	if len(p) > 0 && p[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}
