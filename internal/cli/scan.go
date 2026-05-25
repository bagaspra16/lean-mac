package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/bagaspra16/lean-mac/internal/detectors"
	"github.com/bagaspra16/lean-mac/internal/reporting"
	"github.com/bagaspra16/lean-mac/internal/scanner"
	"github.com/bagaspra16/lean-mac/internal/types"
	"github.com/spf13/cobra"
)

func scanCmd() *cobra.Command {
	var asJSON, asMarkdown bool
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "scan disk and print reclaimable artifacts (no deletion)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rpt := runScan()
			switch {
			case asJSON:
				return reporting.ScanJSON(os.Stdout, rpt)
			case asMarkdown:
				reporting.ScanMarkdown(os.Stdout, rpt)
			default:
				reporting.ScanText(os.Stdout, rpt)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON")
	cmd.Flags().BoolVar(&asMarkdown, "markdown", false, "emit Markdown")
	return cmd
}

func runScan() *types.ScanReport {
	s := scanner.New(detectors.Default()...)
	ch := make(chan scanner.Progress, 64)
	done := make(chan *types.ScanReport, 1)
	go func() { done <- s.Run(context.Background(), ch) }()
	for range ch {
	}
	rpt := <-done
	fmt.Fprintf(os.Stderr, "scan complete: %d findings in %dms\n", len(rpt.Findings), rpt.DurationMS)
	return rpt
}
