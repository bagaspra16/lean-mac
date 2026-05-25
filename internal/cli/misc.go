package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/bagaspra16/lean-mac/internal/detectors"
	"github.com/bagaspra16/lean-mac/internal/fsutil"
	"github.com/bagaspra16/lean-mac/internal/monitor"
	"github.com/bagaspra16/lean-mac/internal/reporting"
	"github.com/bagaspra16/lean-mac/internal/scanner"
	"github.com/bagaspra16/lean-mac/internal/ui"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
)

func tuiCmd() *cobra.Command {
	var dry bool
	c := &cobra.Command{
		Use:   "tui",
		Short: "launch the interactive TUI (default action)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(dry)
		},
	}
	c.Flags().BoolVar(&dry, "dry-run", true, "deletions are simulated (default true for TUI)")
	return c
}

func runTUI(dryRun bool) error {
	s := scanner.New(detectors.Default()...)
	p := tea.NewProgram(ui.NewModel(s, dryRun), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func monitorCmd() *cobra.Command {
	var interval time.Duration
	var warnGB, critGB int64
	c := &cobra.Command{
		Use:   "monitor",
		Short: "stream live disk pressure (Ctrl-C to stop)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sig
				cancel()
			}()
			t := monitor.Thresholds{
				Warn:     warnGB << 30,
				Critical: critGB << 30,
			}
			return monitor.Run(ctx, os.Stdout, t, interval)
		},
	}
	c.Flags().DurationVar(&interval, "interval", 5*time.Second, "poll interval")
	c.Flags().Int64Var(&warnGB, "warn-gb", 25, "warn below this many GB free")
	c.Flags().Int64Var(&critGB, "critical-gb", 10, "critical below this many GB free")
	return c
}

func reportCmd() *cobra.Command {
	var format string
	var outPath string
	c := &cobra.Command{
		Use:   "report",
		Short: "produce a scan report in text/json/markdown",
		RunE: func(cmd *cobra.Command, args []string) error {
			rpt := runScan()
			out := os.Stdout
			if outPath != "" {
				f, err := os.Create(outPath)
				if err != nil {
					return err
				}
				defer f.Close()
				out = f
			}
			switch format {
			case "json":
				return reporting.ScanJSON(out, rpt)
			case "markdown", "md":
				reporting.ScanMarkdown(out, rpt)
			default:
				reporting.ScanText(out, rpt)
			}
			return nil
		},
	}
	c.Flags().StringVarP(&format, "format", "f", "text", "text | json | markdown")
	c.Flags().StringVarP(&outPath, "out", "o", "", "write to file instead of stdout")
	return c
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "check environment and available external tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			tools := []string{"docker", "xcrun", "brew", "git"}
			fmt.Println("lean-mac doctor")
			fmt.Println()
			for _, t := range tools {
				path, err := exec.LookPath(t)
				if err != nil {
					fmt.Printf("  ✗ %-8s not found (%s)\n", t, err)
				} else {
					fmt.Printf("  ✓ %-8s %s\n", t, path)
				}
			}
			free, total, _ := fsutil.DiskUsage("/")
			fmt.Printf("\n  disk free %s / %s\n",
				humanize.IBytes(uint64(free)), humanize.IBytes(uint64(total)))
			return nil
		},
	}
}
