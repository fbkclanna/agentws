package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "run -- <command...>",
		Short:              "Run a command from the workspace root",
		DisableFlagParsing: true,
		RunE:               runRun,
	}
	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Root().Flags().GetString("root")

	if len(args) == 0 {
		return fmt.Errorf("usage: agentws run -- <command...>")
	}

	// Strip leading "--" if present.
	if args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return fmt.Errorf("no command specified after --")
	}

	c := exec.Command(args[0], args[1:]...)
	c.Dir = root
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
