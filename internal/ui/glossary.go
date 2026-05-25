package ui

import "github.com/bagaspra16/lean-mac/internal/types"

// categoryGlossary maps each detector category to a short, user-facing
// description. Used by Help and as inline blurbs in the Scan / AI views.
var categoryGlossary = map[types.Category]string{
	types.CatNode:         "Node project dependencies; reinstall with npm/pnpm/yarn install.",
	types.CatNpmCache:     "npm download cache. Auto-rebuilds on next install.",
	types.CatPnpmStore:    "pnpm content-addressable store. Auto-rebuilds.",
	types.CatYarnCache:    "Yarn package cache. Auto-rebuilds.",
	types.CatPipCache:     "pip wheel cache. Auto-rebuilds.",
	types.CatRustTarget:   "Cargo build output (target/). Recompiles on next build.",
	types.CatRustCargo:    "Cargo registry cache. Large to redownload.",
	types.CatGoCache:      "Go build cache. Auto-rebuilds.",
	types.CatGoModCache:   "Go module cache. Large download to restore.",
	types.CatXcodeDD:      "Xcode intermediate build products (DerivedData).",
	types.CatXcodeArch:    "Xcode archives. Hold shipped builds + dSYMs.",
	types.CatXcodeSim:     "iOS simulator devices (per-device). Recreatable via simctl.",
	types.CatDockerImg:    "Dangling Docker images.",
	types.CatDockerVol:    "Unused Docker volumes — MAY contain database data.",
	types.CatDockerBuild:  "Docker BuildKit cache. Rebuilds on next docker build.",
	types.CatBrewCache:    "Homebrew download cache. Equivalent to brew cleanup.",
	types.CatGradle:       "Gradle dependency cache. Auto-rebuilds.",
	types.CatMaven:        "Maven local repository. Large to redownload.",
}

func categoryBlurb(c types.Category) string {
	if s, ok := categoryGlossary[c]; ok {
		return s
	}
	return ""
}

// riskBlurb describes a risk tier in one user-facing sentence.
func riskBlurb(label string) string {
	switch label {
	case "SAFE":
		return "Caches that auto-regenerate. No project state lost."
	case "MEDIUM":
		return "Reinstall or rebuild required. Costs time and bandwidth."
	case "DANGEROUS":
		return "May lose shipped artifacts, databases, or user data."
	}
	return ""
}
