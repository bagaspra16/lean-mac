package detectors

import (
	"os"

	"github.com/bagaspra16/lean-mac/internal/scanner"
)

// Default returns the production detector set, rooted at the user's home.
func Default() []scanner.Detector {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dets := []scanner.Detector{
		NodeModules{Root: home},
		RustTarget{Root: home},
		Docker{},
		Simulators{},
	}
	for _, p := range DefaultPathDetectors() {
		dets = append(dets, p)
	}
	return dets
}
