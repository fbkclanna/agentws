package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [url ...]",
		Short: "Add repositories to the workspace",
		RunE:  runAdd,
	}
	cmd.Flags().Bool("local", false, "Create a local repository (no remote URL)")
	cmd.Flags().String("id", "", "Repository ID (single URL only)")
	cmd.Flags().String("path", "", "Repository path (single URL only)")
	cmd.Flags().String("ref", "", "Git ref to checkout")
	cmd.Flags().StringSlice("tag", nil, "Tags to assign to the repositories")
	cmd.Flags().Bool("sync", false, "Clone/initialize repositories after adding")
	cmd.Flags().Bool("json", false, "Output added repositories as JSON")
	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	isLocal, _ := cmd.Flags().GetBool("local")
	idOverride, _ := cmd.Flags().GetString("id")
	pathOverride, _ := cmd.Flags().GetString("path")
	refOverride, _ := cmd.Flags().GetString("ref")
	tags, _ := cmd.Flags().GetStringSlice("tag")
	doSync, _ := cmd.Flags().GetBool("sync")
	asJSON, _ := cmd.Flags().GetBool("json")

	ctx, err := workspace.Load(root)
	if err != nil {
		return err
	}

	var newRepos []manifest.Repo
	if isLocal {
		newRepos, err = collectLocalRepos(args, ctx.Manifest.ReposRoot, pathOverride, refOverride, tags)
	} else {
		newRepos, err = collectNewRepos(ctx, args, idOverride, pathOverride, refOverride, tags)
	}
	if err != nil {
		return err
	}

	if len(newRepos) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No repositories added.")
		return nil
	}

	if err := saveWithNewRepos(ctx, newRepos); err != nil {
		return err
	}

	if err := outputResults(cmd, newRepos, asJSON); err != nil {
		return err
	}

	if doSync {
		syncNewRepos(cmd, ctx, newRepos)
	}

	return nil
}

// collectNewRepos gathers repos to add via interactive or CLI mode.
func collectNewRepos(ctx *workspace.Context, args []string, idOverride, pathOverride, refOverride string, tags []string) ([]manifest.Repo, error) {
	existingIDs := make(map[string]bool, len(ctx.Manifest.Repos))
	for _, r := range ctx.Manifest.Repos {
		existingIDs[r.ID] = true
	}

	if len(args) == 0 {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, fmt.Errorf("no URLs provided and stdin is not a TTY; provide URLs as arguments")
		}
		repos, err := interactiveAddRepos(ctx.Manifest.Name, ctx.Manifest.ReposRoot, existingIDs)
		if err != nil {
			return nil, fmt.Errorf("interactive add: %w", err)
		}
		return repos, nil
	}

	if len(args) > 1 && idOverride != "" {
		return nil, fmt.Errorf("--id can only be used with a single URL")
	}
	if len(args) > 1 && pathOverride != "" {
		return nil, fmt.Errorf("--path can only be used with a single URL")
	}
	return buildNewRepos(args, ctx.Manifest.ReposRoot, idOverride, pathOverride, refOverride, tags)
}

// saveWithNewRepos validates and writes the manifest with new repos appended.
func saveWithNewRepos(ctx *workspace.Context, newRepos []manifest.Repo) error {
	if err := findConflicts(ctx.Manifest.Repos, newRepos); err != nil {
		return err
	}

	ctx.Manifest.Repos = append(ctx.Manifest.Repos, newRepos...)
	if err := manifest.Validate(ctx.Manifest); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	return manifest.Save(ctx.ManifestPath, ctx.Manifest)
}

// outputResults prints added repos in text or JSON format.
func outputResults(cmd *cobra.Command, newRepos []manifest.Repo, asJSON bool) error {
	out := cmd.OutOrStdout()
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(newRepos)
	}
	for _, r := range newRepos {
		if r.IsLocal() {
			_, _ = fmt.Fprintf(out, "Added %s (local)\n", r.ID)
		} else {
			_, _ = fmt.Fprintf(out, "Added %s (%s)\n", r.ID, r.URL)
		}
	}
	return nil
}

// syncNewRepos clones or initializes newly added repos.
func syncNewRepos(cmd *cobra.Command, ctx *workspace.Context, newRepos []manifest.Repo) {
	for _, r := range newRepos {
		dir := filepath.Join(ctx.Root, r.Path)
		if git.IsCloned(dir) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Skipping %s (already cloned)\n", r.ID)
			continue
		}
		if r.IsLocal() {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Initializing %s ...\n", r.ID)
			if err := initLocalRepo(dir); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to initialize %s: %v\n", r.ID, err)
			}
			continue
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Cloning %s ...\n", r.ID)
		opts := git.CloneOpts{
			Depth:        r.EffectiveDepth(ctx.Manifest.Defaults),
			PartialClone: r.EffectivePartialClone(ctx.Manifest.Defaults),
			Sparse:       r.Sparse,
		}
		if err := git.Clone(r.URL, dir, opts); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to clone %s: %v (use 'agentws sync' to retry)\n", r.ID, err)
		}
	}
}

// buildNewRepos constructs Repo entries from CLI arguments.
func buildNewRepos(urls []string, reposRoot, idOverride, pathOverride, refOverride string, tags []string) ([]manifest.Repo, error) {
	repos := make([]manifest.Repo, 0, len(urls))
	seen := make(map[string]bool, len(urls))

	for _, u := range urls {
		if u == "" {
			return nil, fmt.Errorf("empty URL is not allowed")
		}

		id := idOverride
		if id == "" {
			id = repoIDFromURL(u)
		}
		if id == "" || id == "." {
			return nil, fmt.Errorf("cannot infer repository ID from URL %q", u)
		}

		if seen[id] {
			return nil, fmt.Errorf("duplicate repository ID %q in arguments", id)
		}
		seen[id] = true

		repoPath := pathOverride
		if repoPath == "" {
			if reposRoot != "" {
				repoPath = reposRoot + "/" + id
			} else {
				repoPath = id
			}
		}

		ref := refOverride
		if ref == "" {
			if b, err := git.DefaultBranch(u); err == nil {
				ref = b
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to detect default branch for %s (%v), using \"main\"\n", u, err)
				ref = "main"
			}
		}

		var repoTags []string
		if len(tags) > 0 {
			repoTags = make([]string, len(tags))
			copy(repoTags, tags)
		}

		repos = append(repos, manifest.Repo{
			ID:      id,
			URL:     u,
			Path:    repoPath,
			Ref:     ref,
			BaseRef: ref,
			Tags:    repoTags,
		})
	}

	return repos, nil
}

// collectLocalRepos validates args for --local mode and builds local repo entries.
func collectLocalRepos(args []string, reposRoot, pathOverride, refOverride string, tags []string) ([]manifest.Repo, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("--local requires at least one repository ID as argument")
	}
	return buildLocalRepos(args, reposRoot, pathOverride, refOverride, tags)
}

// buildLocalRepos constructs local Repo entries from IDs.
func buildLocalRepos(ids []string, reposRoot, pathOverride, refOverride string, tags []string) ([]manifest.Repo, error) {
	if len(ids) > 1 && pathOverride != "" {
		return nil, fmt.Errorf("--path can only be used with a single repository")
	}

	repos := make([]manifest.Repo, 0, len(ids))
	seen := make(map[string]bool, len(ids))

	for _, id := range ids {
		if id == "" {
			return nil, fmt.Errorf("empty repository ID is not allowed")
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate repository ID %q in arguments", id)
		}
		seen[id] = true

		repoPath := pathOverride
		if repoPath == "" {
			if reposRoot != "" {
				repoPath = reposRoot + "/" + id
			} else {
				repoPath = id
			}
		}

		ref := refOverride
		if ref == "" {
			ref = "main"
		}

		var repoTags []string
		if len(tags) > 0 {
			repoTags = make([]string, len(tags))
			copy(repoTags, tags)
		}

		repos = append(repos, manifest.Repo{
			ID:    id,
			Local: true,
			Path:  repoPath,
			Ref:   ref,
			Tags:  repoTags,
		})
	}

	return repos, nil
}

// findConflicts checks for ID or path conflicts between existing and new repos.
func findConflicts(existing, newRepos []manifest.Repo) error {
	idSet := make(map[string]bool, len(existing))
	pathSet := make(map[string]bool, len(existing))
	for _, r := range existing {
		idSet[r.ID] = true
		pathSet[r.Path] = true
	}

	for _, r := range newRepos {
		if idSet[r.ID] {
			return fmt.Errorf("repository ID %q already exists in workspace", r.ID)
		}
		if pathSet[r.Path] {
			return fmt.Errorf("repository path %q already exists in workspace", r.Path)
		}
	}
	return nil
}
