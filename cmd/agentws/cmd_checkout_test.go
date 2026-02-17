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
