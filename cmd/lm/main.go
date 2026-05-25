package main

import (
	"fmt"
	"os"

	"github.com/bagaspra16/lean-mac/internal/cli"
	"github.com/bagaspra16/lean-mac/internal/ui"
)

// version is injected via -ldflags at build time. See Makefile.
var version = "dev"

func main() {
	ui.Version = version
	if err := cli.Root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
