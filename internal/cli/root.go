package cli

import "github.com/spf13/cobra"

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:   "lm",
		Short: "lean-mac — developer-aware storage analysis & cleanup for macOS",
		Long: `lean-mac is a terminal-native storage intelligence platform for developers on macOS.
It detects, attributes, and (with confirmation) reclaims space from build artifacts,
package caches, Docker objects, Xcode data, and iOS simulators.

Run without arguments to launch the interactive TUI.`,
		SilenceUsage: true,
	}
	root.AddCommand(
		scanCmd(),
		cleanCmd(),
		monitorCmd(),
		reportCmd(),
		doctorCmd(),
		tuiCmd(),
	)
	// Default action: launch TUI.
	root.RunE = func(cmd *cobra.Command, args []string) error {
		return runTUI(false)
	}
	return root
}
