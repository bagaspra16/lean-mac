package detectors

import (
	"context"

	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/bagaspra16/lean-mac/internal/types"
)

// PathDetector emits a single Finding for a well-known path if it exists.
// Used for caches whose location is deterministic.
type PathDetector struct {
	N        string
	Category types.Category
	Path     string
	Risk     types.Risk
	Desc     string
}

func (p PathDetector) Name() string { return p.N }

func (p PathDetector) Detect(ctx context.Context, emit func(types.Finding)) error {
	path := fsutil.ExpandHome(p.Path)
	if !fsutil.Exists(path) {
		return nil
	}
	if f := finding(ctx, p.Category, path, p.Risk, p.Desc); f != nil && f.Size > 0 {
		emit(*f)
	}
	return nil
}

// DefaultPathDetectors returns the static-path detectors that apply on macOS.
func DefaultPathDetectors() []PathDetector {
	return []PathDetector{
		{N: "npm-cache", Category: types.CatNpmCache, Path: "~/.npm",
			Risk: types.RiskSafe, Desc: "npm download cache; regenerated automatically."},
		{N: "pnpm-store", Category: types.CatPnpmStore, Path: "~/Library/pnpm/store",
			Risk: types.RiskSafe, Desc: "pnpm content-addressable store; rebuilds on install."},
		{N: "yarn-cache", Category: types.CatYarnCache, Path: "~/Library/Caches/Yarn",
			Risk: types.RiskSafe, Desc: "Yarn package cache; regenerated automatically."},
		{N: "pip-cache", Category: types.CatPipCache, Path: "~/Library/Caches/pip",
			Risk: types.RiskSafe, Desc: "pip wheel cache; regenerated automatically."},
		{N: "go-cache", Category: types.CatGoCache, Path: "~/Library/Caches/go-build",
			Risk: types.RiskSafe, Desc: "Go build cache; regenerated on next build."},
		{N: "go-modcache", Category: types.CatGoModCache, Path: "~/go/pkg/mod",
			Risk: types.RiskMedium, Desc: "Go module cache; large download to restore."},
		{N: "cargo-cache", Category: types.CatRustCargo, Path: "~/.cargo/registry",
			Risk: types.RiskMedium, Desc: "Cargo registry cache; rebuilds on next cargo command."},
		{N: "xcode-deriveddata", Category: types.CatXcodeDD, Path: "~/Library/Developer/Xcode/DerivedData",
			Risk: types.RiskSafe, Desc: "Xcode intermediate build products; safe to remove."},
		{N: "xcode-archives", Category: types.CatXcodeArch, Path: "~/Library/Developer/Xcode/Archives",
			Risk: types.RiskDangerous, Desc: "Xcode archives; deleting loses dSYMs and shipped builds."},
		{N: "homebrew-cache", Category: types.CatBrewCache, Path: "~/Library/Caches/Homebrew",
			Risk: types.RiskSafe, Desc: "Homebrew download cache; `brew cleanup` equivalent."},
		{N: "gradle-cache", Category: types.CatGradle, Path: "~/.gradle/caches",
			Risk: types.RiskSafe, Desc: "Gradle dependency cache; regenerated on next build."},
		{N: "maven-cache", Category: types.CatMaven, Path: "~/.m2/repository",
			Risk: types.RiskMedium, Desc: "Maven local repository; large to redownload."},
	}
}
