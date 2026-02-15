package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/lock"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/ui"
	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Clone/fetch/checkout repos to match the manifest",
		RunE:  runSync,
	}
	cmd.Flags().String("profile", "", "Sync only repos matching the profile")
	cmd.Flags().StringSlice("only", nil, "Sync only these repo IDs")
	cmd.Flags().StringSlice("skip", nil, "Skip these repo IDs")
	cmd.Flags().Int("jobs", 4, "Number of parallel sync workers")
	cmd.Flags().String("strategy", "safe", "Dirty tree strategy: safe, stash, reset")
	cmd.Flags().Bool("force", false, "Allow destructive operations")
	cmd.Flags().Bool("lock", false, "Checkout commits from the lock file")
	cmd.Flags().Bool("update-lock", false, "Update the lock file after sync")
	return cmd
}

func runSync(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("root")
	profile, _ := cmd.Flags().GetString("profile")
	only, _ := cmd.Flags().GetStringSlice("only")
	skip, _ := cmd.Flags().GetStringSlice("skip")
	jobs, _ := cmd.Flags().GetInt("jobs")
	strategyStr, _ := cmd.Flags().GetString("strategy")
	force, _ := cmd.Flags().GetBool("force")
	useLock, _ := cmd.Flags().GetBool("lock")
	updateLock, _ := cmd.Flags().GetBool("update-lock")

	strategy, err := workspace.ParseStrategy(strategyStr)
	if err != nil {
		return err
	}

	if jobs < 1 {
		return fmt.Errorf("--jobs must be >= 1 (got %d)", jobs)
	}

	if strategy == workspace.StrategyReset && !force {
		return fmt.Errorf("--strategy reset requires --force")
	}

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

	if useLock && ctx.Lock == nil {
		return fmt.Errorf("--lock specified but no workspace.lock.yaml found")
	}

	progress := ui.NewProgress(cmd.ErrOrStderr(), len(repos))
	if err := runParallelSync(ctx, repos, strategy, useLock, jobs, progress); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if updateLock {
		if err := writeLock(ctx, repos); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(out, "Lock file updated.")
	}

	_, _ = fmt.Fprintln(out, "Sync complete.")
	return nil
}

func runParallelSync(ctx *workspace.Context, repos []manifest.Repo, strategy workspace.Strategy, useLock bool, jobs int, progress *ui.Progress) error {
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	errCh := make(chan error, len(repos))

	for _, r := range repos {
		wg.Add(1)
		go func(r manifest.Repo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := syncRepo(ctx, r, strategy, useLock, progress); err != nil {
				if r.IsRequired() {
					errCh <- fmt.Errorf("repo %s: %w", r.ID, err)
				} else {
					progress.Log("Warning: optional repo %s: %v", r.ID, err)
				}
			}
		}(r)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		return e
	}
	return nil
}

func syncRepo(ctx *workspace.Context, r manifest.Repo, strategy workspace.Strategy, useLock bool, progress *ui.Progress) error {
	dir := ctx.RepoDir(r)

	if err := cloneOrFetch(dir, r, ctx.Manifest.Defaults, progress); err != nil {
		return err
	}

	skipped, err := handleDirtyForSync(dir, r, strategy, progress)
	if err != nil {
		return err
	}
	if skipped {
		return nil
	}

	// Determine target ref.
	ref := r.EffectiveRef()
	if useLock && ctx.Lock != nil {
		if lr, ok := ctx.Lock.Repos[r.ID]; ok {
			ref = lr.Commit
		}
	}

	if err := git.Checkout(dir, ref); err != nil {
		return fmt.Errorf("checkout %s: %w", ref, err)
	}

	// Run post_sync commands.
	if err := runPostSync(dir, r.PostSync); err != nil {
		return err
	}

	progress.Done(fmt.Sprintf("%s synced @ %s", r.ID, ref))
	return nil
}

func cloneOrFetch(dir string, r manifest.Repo, defaults manifest.Defaults, progress *ui.Progress) error {
	if !git.IsCloned(dir) {
		progress.Log("Cloning %s ...", r.ID)
		opts := git.CloneOpts{
			Depth:        r.EffectiveDepth(defaults),
			PartialClone: r.EffectivePartialClone(defaults),
			Sparse:       r.Sparse,
		}
		return git.Clone(r.URL, dir, opts)
	}
	progress.Log("Fetching %s ...", r.ID)
	if err := git.Fetch(dir); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	return nil
}

func handleDirtyForSync(dir string, r manifest.Repo, strategy workspace.Strategy, progress *ui.Progress) (skipped bool, err error) {
	dirty, err := git.IsDirty(dir)
	if err != nil {
		return false, fmt.Errorf("checking dirty state: %w", err)
	}
	if !dirty {
		return false, nil
	}
	switch strategy {
	case workspace.StrategySafe:
		progress.Done(fmt.Sprintf("%s skipped (dirty)", r.ID))
		return true, nil
	case workspace.StrategyStash:
		progress.Log("Stashing %s ...", r.ID)
		if err := git.Stash(dir); err != nil {
			return false, fmt.Errorf("stash: %w", err)
		}
	case workspace.StrategyReset:
		progress.Log("Resetting %s ...", r.ID)
		if err := git.ResetHard(dir, "HEAD"); err != nil {
			return false, fmt.Errorf("reset: %w", err)
		}
	}
	return false, nil
}

func runPostSync(repoDir string, commands []manifest.PostSync) error {
	for _, ps := range commands {
		fmt.Printf("  Running post_sync: %s\n", ps.Name)
		if err := execCmd(repoDir, ps); err != nil {
			return fmt.Errorf("post_sync %q: %w", ps.Name, err)
		}
	}
	return nil
}

func writeLock(ctx *workspace.Context, repos []manifest.Repo) error {
	lf := &lock.File{
		Version:     1,
		Name:        ctx.Manifest.Name,
		GeneratedAt: time.Now().Format(time.RFC3339),
		ToolVersion: version,
		Repos:       make(map[string]*lock.Repo, len(repos)),
	}
	for _, r := range repos {
		dir := ctx.RepoDir(r)
		if !git.IsCloned(dir) {
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
	}
	return lock.Save(ctx.LockPath, lf)
}
