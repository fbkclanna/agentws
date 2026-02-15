package main

import (
	"fmt"
	"os"
)

// Set via -ldflags at build time.
var version = "dev"

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
