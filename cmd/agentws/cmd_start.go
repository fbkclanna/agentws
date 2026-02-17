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
	cmd.Flags().String("from", "", "Starting point for new branches (overrides base_ref)")
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

	for _, r := range repos {
		if err := startRepo(ctx, r, branch, from, fromExplicit, strategy, dryRun, out); err != nil {
			return err
		}
	}

	return nil
}

func startRepo(ctx *workspace.Context, r manifest.Repo, branch, from string, fromExplicit bool, strategy workspace.Strategy, dryRun bool, out interface{ Write([]byte) (int, error) }) error {
	dir := ctx.RepoDir(r)
	if !git.IsCloned(dir) {
		_, _ = fmt.Fprintf(out, "Skipping %s (not cloned)\n", r.ID)
		return nil
	}

	if err := git.Fetch(dir); err != nil {
		return fmt.Errorf("repo %s: fetch: %w", r.ID, err)
	}

	// Resolve from before handleDirty to avoid side effects (stash/reset)
	// when the resolution fails. Only resolves when branch creation is needed.
	repoFrom, err := resolveFromIfNeeded(dir, branch, true, r, ctx.Manifest.Defaults, from, fromExplicit)
	if err != nil {
		return fmt.Errorf("repo %s: %w", r.ID, err)
	}

	if err := handleDirty(dir, r.ID, strategy); err != nil {
		return err
	}

	action, err := resolveCheckoutAction(dir, branch, true, repoFrom)
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

// resolveFromIfNeeded checks whether the branch needs to be created and, if so,
// resolves the starting point. Called before handleDirty so that config errors
// are returned before any side effects (stash/reset).
func resolveFromIfNeeded(dir, branch string, create bool, r manifest.Repo, d manifest.Defaults, cliFrom string, cliSpecified bool) (string, error) {
	if !create {
		return cliFrom, nil
	}
	localExists, err := git.BranchExists(dir, branch)
	if err != nil {
		return "", err
	}
	if localExists {
		return "", nil
	}
	remoteExists, err := git.RemoteBranchExists(dir, branch)
	if err != nil {
		return "", err
	}
	if remoteExists {
		return "", nil
	}
	return resolveStartFrom(r, d, cliFrom, cliSpecified)
}

func resolveStartFrom(r manifest.Repo, d manifest.Defaults, cliFrom string, cliSpecified bool) (string, error) {
	if cliSpecified {
		return cliFrom, nil
	}
	base := r.EffectiveBaseRef(d)
	if base == "" {
		return "", fmt.Errorf("base_ref is not configured (set base_ref in workspace.yaml or use --from)")
	}
	return "origin/" + base, nil
}

func buildBranchName(prefix, ticket, slug string) string {
	var parts []string
	parts = append(parts, ticket)
	if slug != "" {
		parts = append(parts, slug)
	}
	return prefix + "/" + strings.Join(parts, "-")
}
