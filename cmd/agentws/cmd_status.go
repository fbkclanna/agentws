package main

import (
	"encoding/json"
	"fmt"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/ui"
	"github.com/fbkclanna/agentws/internal/workspace"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show workspace status",
		RunE:  runStatus,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

type repoStatus struct {
	ID       string `json:"id"`
	Local    bool   `json:"local,omitempty"`
	Cloned   bool   `json:"cloned"`
	Branch   string `json:"branch,omitempty"`
	Head     string `json:"head,omitempty"`
	Dirty    bool   `json:"dirty"`
	LockDiff string `json:"lock_diff,omitempty"`
}

func runStatus(cmd *cobra.Command, _ []string) error {
	root, _ := cmd.Flags().GetString("root")
	asJSON, _ := cmd.Flags().GetBool("json")

	ctx, err := workspace.Load(root)
	if err != nil {
		return err
	}

	statuses := make([]repoStatus, 0, len(ctx.Manifest.Repos))
	for _, r := range ctx.Manifest.Repos {
		s := collectStatus(ctx, r)
		statuses = append(statuses, s)
	}

	out := cmd.OutOrStdout()

	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(statuses)
	}

	tbl := ui.NewTable(out, "REPO", "STATE", "BRANCH", "HEAD", "DIRTY", "LOCK DIFF")
	for _, s := range statuses {
		state := "cloned"
		if !s.Cloned {
			state = "not cloned"
		}
		if s.Local {
			state += " (local)"
		}
		tbl.Row(s.ID, state, s.Branch, s.Head, s.Dirty, s.LockDiff)
	}
	return tbl.Flush()
}

func collectStatus(ctx *workspace.Context, r manifest.Repo) repoStatus {
	dir := ctx.RepoDir(r)
	s := repoStatus{ID: r.ID, Local: r.IsLocal()}

	if !git.IsCloned(dir) {
		return s
	}
	s.Cloned = true

	if branch, err := git.CurrentBranch(dir); err == nil {
		if branch == "" {
			s.Branch = "(detached)"
		} else {
			s.Branch = branch
		}
	}
	if head, err := git.HeadCommit(dir); err == nil {
		s.Head = head
	}
	if dirty, err := git.IsDirty(dir); err == nil {
		s.Dirty = dirty
	}

	if ctx.Lock != nil {
		if lr, ok := ctx.Lock.Repos[r.ID]; ok {
			currentFull, _ := git.HeadCommitFull(dir)
			if currentFull != "" && currentFull != lr.Commit {
				s.LockDiff = fmt.Sprintf("lock=%s", lr.Commit[:minLen(len(lr.Commit), 7)])
			}
		}
	}

	return s
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
