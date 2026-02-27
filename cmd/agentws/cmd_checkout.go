package main

import (
	"fmt"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
)

func newCheckoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkout",
		Short: "Switch all repos to the same branch",
		RunE:  runCheckout,
	}
	cmd.Flags().String("branch", "", "Branch name to checkout (required)")
	_ = cmd.MarkFlagRequired("branch")
	cmd.Flags().Bool("create", false, "Create the branch if it does not exist")
	cmd.Flags().String("from", "", "Starting point for new branches (overrides base_ref)")
	cmd.Flags().String("profile", "", "Filter by profile")
	cmd.Flags().StringSlice("only", nil, "Include only these repo IDs")
	cmd.Flags().StringSlice("skip", nil, "Exclude these repo IDs")
	cmd.Flags().String("strategy", "safe", "Dirty tree strategy: safe, stash, reset")
	cmd.Flags().Bool("force", false, "Allow destructive operations")
	cmd.Flags().Bool("dry-run", false, "Show what would happen without making changes")
	return cmd
}

func runCheckout(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("root")
	branch, _ := cmd.Flags().GetString("branch")
	create, _ := cmd.Flags().GetBool("create")
	from, _ := cmd.Flags().GetString("from")
	profile, _ := cmd.Flags().GetString("profile")
	only, _ := cmd.Flags().GetStringSlice("only")
	skip, _ := cmd.Flags().GetStringSlice("skip")
	strategyStr, _ := cmd.Flags().GetString("strategy")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	strategy, err := workspace.ParseStrategy(strategyStr)
	if err != nil {
		return err
	}
	if strategy == workspace.StrategyReset && !force {
		return fmt.Errorf("--strategy reset requires --force")
	}

	fromExplicit := cmd.Flags().Changed("from")

	ctx, err := workspace.Load(root)
	if err != nil {
		return err
	}

	repos := ctx.Manifest.Repos
	if profile != "" {
		repos, err = manifest.FilterRepos(ctx.Manifest, profile)
		if err != nil {
			return err
		}
	}
	repos = manifest.FilterByIDs(repos, only, skip)

	out := cmd.OutOrStdout()
	for _, r := range repos {
		if err := checkoutRepo(ctx, r, branch, create, from, fromExplicit, strategy, dryRun, out); err != nil {
			return err
		}
	}

	return nil
}

func checkoutRepo(ctx *workspace.Context, r manifest.Repo, branch string, create bool, from string, fromExplicit bool, strategy workspace.Strategy, dryRun bool, out interface{ Write([]byte) (int, error) }) error {
	dir := ctx.RepoDir(r)
	if !git.IsCloned(dir) {
		_, _ = fmt.Fprintf(out, "Skipping %s (not cloned)\n", r.ID)
		return nil
	}

	if !r.IsLocal() {
		if err := git.Fetch(dir); err != nil {
			return fmt.Errorf("repo %s: fetch: %w", r.ID, err)
		}
	}

	// Resolve from before handleDirty to avoid side effects (stash/reset)
	// when the resolution fails. Only resolves when branch creation is needed.
	repoFrom, err := resolveFromIfNeeded(dir, branch, create, r, ctx.Manifest.Defaults, from, fromExplicit)
	if err != nil {
		return fmt.Errorf("repo %s: %w", r.ID, err)
	}

	if err := handleDirty(dir, r.ID, strategy); err != nil {
		return err
	}

	action, err := resolveCheckoutAction(dir, branch, create, repoFrom, r.IsLocal())
	if err != nil {
		return fmt.Errorf("repo %s: %w", r.ID, err)
	}

	if dryRun {
		_, _ = fmt.Fprintf(out, "[dry-run] %s: %s\n", r.ID, action.description)
		return nil
	}

	if err := action.execute(dir); err != nil {
		return fmt.Errorf("repo %s: %w", r.ID, err)
	}
	_, _ = fmt.Fprintf(out, "%s: %s\n", r.ID, action.description)
	return nil
}

type checkoutAction struct {
	description string
	execute     func(dir string) error
}

func resolveCheckoutAction(dir, branch string, create bool, from string, isLocal bool) (checkoutAction, error) {
	localExists, err := git.BranchExists(dir, branch)
	if err != nil {
		return checkoutAction{}, err
	}
	if localExists {
		return checkoutAction{
			description: fmt.Sprintf("checkout existing local branch %s", branch),
			execute:     func(dir string) error { return git.Checkout(dir, branch) },
		}, nil
	}

	if !isLocal {
		remoteExists, err := git.RemoteBranchExists(dir, branch)
		if err != nil {
			return checkoutAction{}, err
		}
		if remoteExists {
			return checkoutAction{
				description: fmt.Sprintf("create tracking branch %s from origin", branch),
				execute:     func(dir string) error { return git.CreateTrackingBranch(dir, branch) },
			}, nil
		}
	}

	if create {
		return checkoutAction{
			description: fmt.Sprintf("create new branch %s from %s", branch, from),
			execute:     func(dir string) error { return git.CreateBranch(dir, branch, from) },
		}, nil
	}

	return checkoutAction{
		description: fmt.Sprintf("skip (branch %s not found)", branch),
		execute:     func(_ string) error { return nil },
	}, nil
}

func handleDirty(dir, repoID string, strategy workspace.Strategy) error {
	dirty, err := git.IsDirty(dir)
	if err != nil {
		return fmt.Errorf("checking dirty state for %s: %w", repoID, err)
	}
	if !dirty {
		return nil
	}

	switch strategy {
	case workspace.StrategySafe:
		return fmt.Errorf("repo %s has uncommitted changes (use --strategy stash or reset)", repoID)
	case workspace.StrategyStash:
		return git.Stash(dir)
	case workspace.StrategyReset:
		return git.ResetHard(dir, "HEAD")
	}
	return nil
}
