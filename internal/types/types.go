package types

import "time"

type Risk int

const (
	RiskSafe Risk = iota
	RiskMedium
	RiskDangerous
)

func (r Risk) String() string {
	switch r {
	case RiskSafe:
		return "SAFE"
	case RiskMedium:
		return "MEDIUM"
	case RiskDangerous:
		return "DANGEROUS"
	}
	return "?"
}

type Category string

const (
	CatNode       Category = "node_modules"
	CatNpmCache   Category = "npm-cache"
	CatPnpmStore  Category = "pnpm-store"
	CatYarnCache  Category = "yarn-cache"
	CatPipCache   Category = "pip-cache"
	CatRustTarget Category = "rust-target"
	CatRustCargo  Category = "cargo-cache"
	CatGoCache    Category = "go-cache"
	CatGoModCache Category = "go-modcache"
	CatXcodeDD    Category = "xcode-deriveddata"
	CatXcodeArch  Category = "xcode-archives"
	CatXcodeSim   Category = "ios-simulators"
	CatDockerImg  Category = "docker-images"
	CatDockerVol  Category = "docker-volumes"
	CatDockerBuild Category = "docker-buildcache"
	CatBrewCache  Category = "homebrew-cache"
	CatGradle     Category = "gradle-cache"
	CatMaven      Category = "maven-cache"
)

// Finding is a single reclaimable artifact discovered by a Detector.
type Finding struct {
	Category    Category  `json:"category"`
	Path        string    `json:"path"`
	Size        int64     `json:"size_bytes"`
	LastMod     time.Time `json:"last_modified"`
	Risk        Risk      `json:"-"`
	RiskLabel   string    `json:"risk"`
	Reversible  bool      `json:"reversible"`
	Description string    `json:"description"`
	// External means the bytes are not reclaimable via plain rm
	// (e.g. Docker objects deleted via `docker` CLI).
	External bool `json:"external"`
}

type ScanReport struct {
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	HostPath    string    `json:"host_path"`
	Findings    []Finding `json:"findings"`
	TotalBytes  int64     `json:"total_reclaimable_bytes"`
	DiskFree    int64     `json:"disk_free_bytes"`
	DiskTotal   int64     `json:"disk_total_bytes"`
	DurationMS  int64     `json:"duration_ms"`
}

type CleanResult struct {
	Finding      Finding `json:"finding"`
	Success      bool    `json:"success"`
	BytesFreed   int64   `json:"bytes_freed"`
	Error        string  `json:"error,omitempty"`
	DryRun       bool    `json:"dry_run"`
}

type CleanReport struct {
	StartedAt   time.Time     `json:"started_at"`
	FinishedAt  time.Time     `json:"finished_at"`
	Results     []CleanResult `json:"results"`
	BytesFreed  int64         `json:"bytes_freed"`
	DryRun      bool          `json:"dry_run"`
	DiskBefore  int64         `json:"disk_free_before"`
	DiskAfter   int64         `json:"disk_free_after"`
}
