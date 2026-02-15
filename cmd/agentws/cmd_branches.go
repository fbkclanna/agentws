package main

import (
	"encoding/json"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/ui"
	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
)

func newBranchesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branches",
		Short: "List branches, HEAD, and dirty state of each repo",
		RunE:  runBranches,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().String("profile", "", "Filter by profile")
	cmd.Flags().StringSlice("only", nil, "Include only these repo IDs")
	cmd.Flags().StringSlice("skip", nil, "Exclude these repo IDs")
	return cmd
}

type branchInfo struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Head   string `json:"head"`
	Dirty  bool   `json:"dirty"`
}

func runBranches(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("root")
	asJSON, _ := cmd.Flags().GetBool("json")
	profile, _ := cmd.Flags().GetString("profile")
	only, _ := cmd.Flags().GetStringSlice("only")
	skip, _ := cmd.Flags().GetStringSlice("skip")

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

	infos := make([]branchInfo, 0, len(repos))
	for _, r := range repos {
		dir := ctx.RepoDir(r)
		bi := branchInfo{Repo: r.ID}

		if !git.IsCloned(dir) {
			bi.Branch = "(not cloned)"
			infos = append(infos, bi)
			continue
		}

		branch, err := git.CurrentBranch(dir)
		if err != nil {
			bi.Branch = "(error)"
		} else if branch == "" {
			bi.Branch = "(detached)"
		} else {
			bi.Branch = branch
		}
		head, err := git.HeadCommit(dir)
		if err == nil {
			bi.Head = head
		}
		dirty, err := git.IsDirty(dir)
		if err == nil {
			bi.Dirty = dirty
		}
		infos = append(infos, bi)
	}

	out := cmd.OutOrStdout()

	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(infos)
	}

	tbl := ui.NewTable(out, "REPO", "BRANCH", "HEAD", "DIRTY")
	for _, bi := range infos {
		tbl.Row(bi.Repo, bi.Branch, bi.Head, bi.Dirty)
	}
	return tbl.Flush()
}
