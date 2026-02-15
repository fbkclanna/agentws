package main

import (
	"fmt"
	"time"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/lock"
	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
)

func newPinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pin",
		Short: "Pin current HEAD commits to the lock file",
		RunE:  runPin,
	}
}

func runPin(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("root")

	ctx, err := workspace.Load(root)
	if err != nil {
		return err
	}

	lf := &lock.File{
		Version:     1,
		Name:        ctx.Manifest.Name,
		GeneratedAt: time.Now().Format(time.RFC3339),
		ToolVersion: version,
		Repos:       make(map[string]*lock.Repo, len(ctx.Manifest.Repos)),
	}

	for _, r := range ctx.Manifest.Repos {
		dir := ctx.RepoDir(r)
		if !git.IsCloned(dir) {
			fmt.Printf("Skipping %s (not cloned)\n", r.ID)
			continue
		}
		commit, err := git.HeadCommitFull(dir)
		if err != nil {
			return fmt.Errorf("reading HEAD for %s: %w", r.ID, err)
		}
		lf.Repos[r.ID] = &lock.Repo{
			URL:    r.URL,
			Ref:    r.EffectiveRef(),
			Commit: commit,
		}
		fmt.Printf("Pinned %s @ %s\n", r.ID, commit[:minLen(len(commit), 7)])
	}

	if err := lock.Save(ctx.LockPath, lf); err != nil {
		return err
	}

	fmt.Printf("Lock file written to %s\n", ctx.LockPath)
	return nil
}
