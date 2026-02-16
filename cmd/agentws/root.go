package main

import (
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agentws",
		Short:   "Multi-repo workspace manager for coding agents",
		Version: version,
	}

	cmd.PersistentFlags().String("root", ".", "Root directory for workspaces")

	cmd.AddCommand(
		newInitCmd(),
		newAddCmd(),
		newSyncCmd(),
		newStatusCmd(),
		newPinCmd(),
		newBranchesCmd(),
		newCheckoutCmd(),
		newStartCmd(),
		newDoctorCmd(),
		newRunCmd(),
	)

	return cmd
}
