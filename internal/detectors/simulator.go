package detectors

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/bagaspra16/lean-mac/internal/types"
)

// Simulators sizes the iOS simulator devices directory and emits one finding
// per shutdown device. Booted devices are skipped.
type Simulators struct{}

func (Simulators) Name() string { return "ios-simulators" }

type simDevice struct {
	UDID  string `json:"udid"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type simctlList struct {
	Devices map[string][]simDevice `json:"devices"`
}

func (Simulators) Detect(ctx context.Context, emit func(types.Finding)) error {
	if _, err := exec.LookPath("xcrun"); err != nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, "xcrun", "simctl", "list", "devices", "--json").Output()
	if err != nil {
		return nil
	}
	var list simctlList
	if err := json.Unmarshal(out, &list); err != nil {
		return nil
	}
	devicesDir := fsutil.ExpandHome("~/Library/Developer/CoreSimulator/Devices")
	for _, devs := range list.Devices {
		for _, dev := range devs {
			if dev.State == "Booted" {
				continue
			}
			path := filepath.Join(devicesDir, dev.UDID)
			if !fsutil.Exists(path) {
				continue
			}
			size, _ := fsutil.DirSize(ctx, path)
			if size == 0 {
				continue
			}
			emit(types.Finding{
				Category:    types.CatXcodeSim,
				Path:        path,
				Size:        size,
				LastMod:     time.Now(),
				Risk:        types.RiskMedium,
				Description: "Shutdown simulator: " + dev.Name + " (recreated via `xcrun simctl create`).",
			})
		}
	}
	return nil
}
