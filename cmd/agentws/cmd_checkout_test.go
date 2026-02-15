package main

import (
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
)

func TestRunCheckout_createBranch(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// Sync first.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Checkout with --create.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/test", "--create", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout --create failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/test" {
		t.Errorf("expected branch feature/test, got %s", branch)
	}
}

func TestRunCheckout_existingBranch(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Create a branch first.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := git.CreateBranch(dir, "my-branch", "HEAD"); err != nil {
		t.Fatalf("create branch: %v", err)
	}
	if err := git.Checkout(dir, "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	// Checkout existing branch.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "my-branch"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout failed: %v", err)
	}

	branch, _ := git.CurrentBranch(dir)
	if branch != "my-branch" {
		t.Errorf("expected branch my-branch, got %s", branch)
	}
}

func TestRunCheckout_dryRun(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "new-branch", "--create", "--from", "HEAD", "--dry-run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout --dry-run failed: %v", err)
	}

	// Branch should NOT have been created.
	dir := filepath.Join(wsDir, "repos", "backend")
	exists, _ := git.BranchExists(dir, "new-branch")
	if exists {
		t.Error("branch should not be created in dry-run mode")
	}
}
