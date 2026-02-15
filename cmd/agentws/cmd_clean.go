package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove workspace repos (destructive, requires --force)",
		RunE:  runClean,
	}
	cmd.Flags().Bool("force", false, "Required to confirm destructive operation")
	return cmd
}

func runClean(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("root")
	force, _ := cmd.Flags().GetBool("force")

	if !force {
		return fmt.Errorf("clean is destructive; pass --force to confirm")
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	if abs == string(filepath.Separator) {
		return fmt.Errorf("refusing to clean root directory: %s", abs)
	}
	if _, err := os.Stat(filepath.Join(abs, "workspace.yaml")); err != nil {
		return fmt.Errorf("refusing to clean %s: no workspace.yaml found (not a workspace directory)", abs)
	}

	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("removing workspace: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workspace removed: %s\n", abs)
	return nil
}
