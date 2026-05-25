package detectors

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/bagaspra16/lean-mac/internal/types"
)

// Docker emits findings derived from `docker system df`. It never enumerates
// containers/volumes individually (that's the cleaner's job); it just reports
// the three top-level reclaimable buckets so the user can decide.
type Docker struct{}

func (Docker) Name() string { return "docker" }

type dockerDFItem struct {
	Type        string `json:"Type"`
	Reclaimable string `json:"Reclaimable"`
	Size        string `json:"Size"`
}

func (Docker) Detect(ctx context.Context, emit func(types.Finding)) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, "docker", "system", "df", "--format", "{{json .}}").Output()
	if err != nil {
		return nil // docker daemon not running; not fatal
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var item dockerDFItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		bytes := parseReclaimable(item.Reclaimable)
		if bytes == 0 {
			continue
		}
		var cat types.Category
		var risk types.Risk
		var desc string
		switch item.Type {
		case "Images":
			cat, risk = types.CatDockerImg, types.RiskMedium
			desc = "Dangling/unused Docker images; `docker image prune -a` equivalent."
		case "Containers":
			continue // not reclaimable in the same sense
		case "Local Volumes":
			cat, risk = types.CatDockerVol, types.RiskDangerous
			desc = "Unused Docker volumes; MAY contain database data."
		case "Build Cache":
			cat, risk = types.CatDockerBuild, types.RiskSafe
			desc = "Docker BuildKit cache; rebuilds on next image build."
		default:
			continue
		}
		emit(types.Finding{
			Category:    cat,
			Path:        "docker://" + item.Type,
			Size:        bytes,
			LastMod:     time.Now(),
			Risk:        risk,
			Description: desc,
			External:    true,
		})
	}
	return nil
}

// parseReclaimable handles formats like "1.234GB (45%)", "512MB", "0B".
func parseReclaimable(s string) int64 {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "("); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" || s == "0B" {
		return 0
	}
	units := map[string]float64{
		"B": 1, "KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12,
		"KiB": 1024, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40,
	}
	// split number / unit
	var numEnd int
	for i, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			numEnd = i + 1
			continue
		}
		break
	}
	if numEnd == 0 {
		return 0
	}
	numStr := s[:numEnd]
	unit := strings.TrimSpace(s[numEnd:])
	var num float64
	for _, c := range numStr {
		if c >= '0' && c <= '9' {
			num = num*10 + float64(c-'0')
		} else if c == '.' {
			// crude: switch to fractional parsing
			frac := 0.0
			place := 0.1
			rest := numStr[strings.Index(numStr, ".")+1:]
			for _, c := range rest {
				if c >= '0' && c <= '9' {
					frac += float64(c-'0') * place
					place *= 0.1
				}
			}
			num += frac
			break
		}
	}
	mult, ok := units[unit]
	if !ok {
		return 0
	}
	return int64(num * mult)
}
