package main

import (
	"fmt"
	"strings"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <ticket> [slug]",
		Short: "Create and checkout a feature branch across repos",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runStart,
	}
	cmd.Flags().String("prefix", "feature", "Branch prefix: feature, bugfix, hotfix")
	cmd.Flags().String("from", "origin/main", "Starting point for new branches")
	cmd.Flags().String("profile", "", "Filter by profile")
	cmd.Flags().StringSlice("only", nil, "Include only these repo IDs")
	cmd.Flags().StringSlice("skip", nil, "Exclude these repo IDs")
	cmd.Flags().String("strategy", "safe", "Dirty tree strategy: safe, stash, reset")
	cmd.Flags().Bool("force", false, "Allow destructive operations")
	cmd.Flags().Bool("dry-run", false, "Show what would happen without making changes")
	return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	prefix, _ := cmd.Flags().GetString("prefix")
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

	ticket := args[0]
	slug := ""
	if len(args) > 1 {
		slug = args[1]
	}

	branch := buildBranchName(prefix, ticket, slug)
	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "Branch: %s\n", branch)

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

	for _, r := range repos {
		dir := ctx.RepoDir(r)
		if !git.IsCloned(dir) {
			_, _ = fmt.Fprintf(out, "Skipping %s (not cloned)\n", r.ID)
			continue
		}

		if err := git.Fetch(dir); err != nil {
			return fmt.Errorf("repo %s: fetch: %w", r.ID, err)
		}

		if err := handleDirty(dir, r.ID, strategy); err != nil {
			return err
		}

		action, err := resolveCheckoutAction(dir, branch, true, from)
		if err != nil {
			return fmt.Errorf("repo %s: %w", r.ID, err)
		}

		if dryRun {
			_, _ = fmt.Fprintf(out, "[dry-run] %s: %s\n", r.ID, action.description)
			continue
		}

		if err := action.execute(dir); err != nil {
			return fmt.Errorf("repo %s: %w", r.ID, err)
		}
		_, _ = fmt.Fprintf(out, "%s: %s\n", r.ID, action.description)
	}

	return nil
}

func buildBranchName(prefix, ticket, slug string) string {
	var parts []string
	parts = append(parts, ticket)
	if slug != "" {
		parts = append(parts, slug)
	}
	return prefix + "/" + strings.Join(parts, "-")
}
