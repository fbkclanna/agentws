package main

import (
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
)

func TestRunStart_createsBranch(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "ABC-123", "search-v2", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/ABC-123-search-v2" {
		t.Errorf("expected branch feature/ABC-123-search-v2, got %s", branch)
	}
}

func TestRunStart_customPrefix(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "BUG-456", "--prefix", "bugfix", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "bugfix/BUG-456" {
		t.Errorf("expected branch bugfix/BUG-456, got %s", branch)
	}
}

func TestRunStart_dryRun(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "DRY-1", "test", "--from", "HEAD", "--dry-run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start --dry-run failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	exists, _ := git.BranchExists(dir, "feature/DRY-1-test")
	if exists {
		t.Error("branch should not be created in dry-run mode")
	}
}

func TestBuildBranchName(t *testing.T) {
	tests := []struct {
		prefix, ticket, slug, want string
	}{
		{"feature", "ABC-123", "search-v2", "feature/ABC-123-search-v2"},
		{"bugfix", "BUG-456", "", "bugfix/BUG-456"},
		{"hotfix", "HOT-1", "fix", "hotfix/HOT-1-fix"},
	}
	for _, tt := range tests {
		got := buildBranchName(tt.prefix, tt.ticket, tt.slug)
		if got != tt.want {
			t.Errorf("buildBranchName(%q, %q, %q) = %q, want %q", tt.prefix, tt.ticket, tt.slug, got, tt.want)
		}
	}
}
