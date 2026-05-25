package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bagaspra16/lean-mac/internal/cleaner"
	"github.com/bagaspra16/lean-mac/internal/reporting"
	"github.com/bagaspra16/lean-mac/internal/types"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func cleanCmd() *cobra.Command {
	var (
		dryRun     bool
		yes        bool
		aggressive bool
		dangerous  bool
		asJSON     bool
		only       []string
	)
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "delete reclaimable artifacts (interactive confirm by default)",
		Long: `clean scans for reclaimable artifacts and, after per-category confirmation,
deletes them. SAFE-risk findings are always eligible; --aggressive includes MEDIUM,
--dangerous includes DANGEROUS. Use --dry-run to preview.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rpt := runScan()
			c := cleaner.New(cleaner.Options{
				DryRun:           dryRun,
				Aggressive:       aggressive,
				IncludeDangerous: dangerous,
			})
			selected := filterEligible(rpt.Findings, c, only)
			if len(selected) == 0 {
				fmt.Fprintln(os.Stderr, "nothing eligible to clean")
				return nil
			}
			if !yes {
				confirmed, err := confirmInteractive(selected, dryRun)
				if err != nil {
					return err
				}
				selected = confirmed
				if len(selected) == 0 {
					fmt.Fprintln(os.Stderr, "no categories confirmed; exiting")
					return nil
				}
			}
			result := c.Clean(context.Background(), selected)
			if asJSON {
				return reporting.CleanJSON(os.Stdout, result)
			}
			printCleanSummary(result)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "do not delete; report what would be removed")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip interactive confirmation")
	cmd.Flags().BoolVar(&aggressive, "aggressive", false, "include MEDIUM risk findings")
	cmd.Flags().BoolVar(&dangerous, "dangerous", false, "include DANGEROUS findings (use with care)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	cmd.Flags().StringSliceVar(&only, "only", nil, "limit to comma-separated categories (e.g. node_modules,xcode-deriveddata)")
	return cmd
}

func filterEligible(in []types.Finding, c *cleaner.Cleaner, only []string) []types.Finding {
	onlySet := map[string]bool{}
	for _, k := range only {
		onlySet[strings.ToLower(strings.TrimSpace(k))] = true
	}
	var out []types.Finding
	for _, f := range in {
		if !c.Eligible(f) {
			continue
		}
		if len(onlySet) > 0 && !onlySet[strings.ToLower(string(f.Category))] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// confirmInteractive asks per-category whether to proceed and returns the
// findings the user kept. Returns immediately if stdin is not a terminal and
// the user must pass --yes instead.
func confirmInteractive(findings []types.Finding, dryRun bool) ([]types.Finding, error) {
	if !termIsInteractive() {
		return nil, fmt.Errorf("non-interactive stdin; pass --yes to confirm without prompting")
	}
	groups := map[types.Category][]types.Finding{}
	for _, f := range findings {
		groups[f.Category] = append(groups[f.Category], f)
	}
	in := bufio.NewReader(os.Stdin)
	mode := "DELETE"
	if dryRun {
		mode = "dry-run"
	}
	fmt.Fprintf(os.Stderr, "\nlean-mac (%s) — review each category:\n\n", mode)
	var kept []types.Finding
	for cat, items := range groups {
		var total int64
		for _, f := range items {
			total += f.Size
		}
		risk := items[0].Risk.String()
		fmt.Fprintf(os.Stderr, "  [%s] %s — %d items, %s\n",
			risk, cat, len(items), humanize.IBytes(uint64(total)))
		for i, f := range items {
			if i >= 3 {
				fmt.Fprintf(os.Stderr, "      … and %d more\n", len(items)-3)
				break
			}
			fmt.Fprintf(os.Stderr, "      • %s  (%s)\n", f.Path, humanize.IBytes(uint64(f.Size)))
		}
		fmt.Fprintf(os.Stderr, "  proceed? [y/N] ")
		ans, _ := in.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans == "y" || ans == "yes" {
			kept = append(kept, items...)
		}
		fmt.Fprintln(os.Stderr)
	}
	return kept, nil
}

func termIsInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func printCleanSummary(r *types.CleanReport) {
	mode := "executed"
	if r.DryRun {
		mode = "dry-run"
	}
	fmt.Fprintf(os.Stderr, "\ncleanup %s: %s reclaimed (%d items)\n",
		mode, humanize.IBytes(uint64(r.BytesFreed)), len(r.Results))
	for _, res := range r.Results {
		status := "✓"
		if !res.Success {
			status = "✗"
		}
		line := fmt.Sprintf("  %s %-22s %10s  %s",
			status, res.Finding.Category,
			humanize.IBytes(uint64(res.BytesFreed)), res.Finding.Path)
		if res.Error != "" {
			line += "  (" + res.Error + ")"
		}
		fmt.Fprintln(os.Stderr, line)
	}
}
