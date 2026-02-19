package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/testutil"
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

func TestRunCheckout_baseRefDefaults(t *testing.T) {
	wsDir := t.TempDir()
	bare := testutil.CreateBareRepo(t)

	wsYAML := fmt.Sprintf("version: 1\nname: test\nrepos_root: repos\ndefaults:\n  base_ref: main\nrepos:\n  - id: backend\n    url: %s\n    path: repos/backend\n    ref: main\n", bare)
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// checkout --create without --from should use defaults.base_ref
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/checkout-base", "--create"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout --create failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/checkout-base" {
		t.Errorf("expected branch feature/checkout-base, got %s", branch)
	}
}

func TestRunCheckout_fromFlagOverridesBaseRef(t *testing.T) {
	wsDir := t.TempDir()
	bare := testutil.CreateBareRepo(t)

	wsYAML := fmt.Sprintf("version: 1\nname: test\nrepos_root: repos\ndefaults:\n  base_ref: nonexistent\nrepos:\n  - id: backend\n    url: %s\n    path: repos/backend\n    ref: main\n", bare)
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// --from HEAD should override base_ref (which points to nonexistent branch)
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/checkout-from", "--create", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout --create --from failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/checkout-from" {
		t.Errorf("expected branch feature/checkout-from, got %s", branch)
	}
}

func TestRunCheckout_noBaseRefNoFromWithCreate(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// --create without --from and no base_ref → should error
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/nobase", "--create"})
	err := root2.Execute()
	if err == nil {
		t.Fatal("expected error when base_ref and --from are both unset with --create")
	}
	if !strings.Contains(err.Error(), "base_ref is not configured") {
		t.Errorf("expected error about base_ref not configured, got: %v", err)
	}
}

func TestRunCheckout_multiRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 3)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/multi", "--create", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout --create failed: %v", err)
	}

	for _, name := range []string{"backend", "frontend", "infra"} {
		dir := filepath.Join(wsDir, "repos", name)
		branch, _ := git.CurrentBranch(dir)
		if branch != "feature/multi" {
			t.Errorf("repo %s: expected branch feature/multi, got %s", name, branch)
		}
	}
}

func TestRunCheckout_remoteBranch(t *testing.T) {
	wsDir := t.TempDir()
	bare := testutil.CreateBareRepoWithBranch(t, "feature/remote")
	wsYAML := fmt.Sprintf("version: 1\nname: test\nrepos_root: repos\nrepos:\n  - id: backend\n    url: %s\n    path: repos/backend\n    ref: main\n", bare)
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/remote"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout remote branch failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/remote" {
		t.Errorf("expected branch feature/remote, got %s", branch)
	}
}

func TestRunCheckout_onlySkipFilters(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/only", "--create", "--from", "HEAD", "--only", "backend"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout --only failed: %v", err)
	}

	backendDir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(backendDir)
	if branch != "feature/only" {
		t.Errorf("backend: expected branch feature/only, got %s", branch)
	}

	frontendDir := filepath.Join(wsDir, "repos", "frontend")
	fbranch, _ := git.CurrentBranch(frontendDir)
	if fbranch != "main" {
		t.Errorf("frontend: expected branch main, got %s", fbranch)
	}
}

func TestRunCheckout_strategySafeDirtyError(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Make dirty.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "newbranch", "--create", "--from", "HEAD"})
	err := root2.Execute()
	if err == nil {
		t.Fatal("expected error for dirty working tree with safe strategy")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("expected error about uncommitted changes, got: %v", err)
	}
}

func TestRunCheckout_strategyStashDirty(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Make dirty.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatal(err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/stashed", "--create", "--from", "HEAD", "--strategy", "stash"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout with stash strategy failed: %v", err)
	}

	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/stashed" {
		t.Errorf("expected branch feature/stashed, got %s", branch)
	}
}

func TestRunCheckout_skipsUncloned(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	// Sync only backend.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--only", "backend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --only backend failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "feature/skip", "--create", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout should skip uncloned repos without error: %v", err)
	}

	backendDir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(backendDir)
	if branch != "feature/skip" {
		t.Errorf("backend: expected branch feature/skip, got %s", branch)
	}
}

func TestRunCheckout_branchNotFoundSkips(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Checkout nonexistent branch without --create should skip.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "checkout", "--branch", "nonexistent"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("checkout nonexistent branch should skip without error: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "main" {
		t.Errorf("expected branch main (unchanged), got %s", branch)
	}
}
