package main

import (
	"fmt"
	"os"

	"github.com/bagaspra16/lean-mac/internal/cli"
)

func main() {
	if err := cli.Root().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
