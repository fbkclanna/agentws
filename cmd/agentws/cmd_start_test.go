package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/testutil"
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

func TestRunStart_multiRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 3)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "MULTI-1", "search", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	for _, name := range []string{"backend", "frontend", "infra"} {
		dir := filepath.Join(wsDir, "repos", name)
		branch, _ := git.CurrentBranch(dir)
		if branch != "feature/MULTI-1-search" {
			t.Errorf("repo %s: expected branch feature/MULTI-1-search, got %s", name, branch)
		}
	}
}

func TestRunStart_onlyFilter(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 3)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "ONLY-1", "api", "--from", "HEAD", "--only", "backend"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start --only failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/ONLY-1-api" {
		t.Errorf("backend: expected branch feature/ONLY-1-api, got %s", branch)
	}

	for _, name := range []string{"frontend", "infra"} {
		d := filepath.Join(wsDir, "repos", name)
		exists, _ := git.BranchExists(d, "feature/ONLY-1-api")
		if exists {
			t.Errorf("repo %s should not have branch feature/ONLY-1-api", name)
		}
	}
}

func TestRunStart_skipFilter(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 3)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "SKIP-1", "web", "--from", "HEAD", "--skip", "infra"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start --skip failed: %v", err)
	}

	for _, name := range []string{"backend", "frontend"} {
		d := filepath.Join(wsDir, "repos", name)
		branch, _ := git.CurrentBranch(d)
		if branch != "feature/SKIP-1-web" {
			t.Errorf("repo %s: expected branch feature/SKIP-1-web, got %s", name, branch)
		}
	}

	exists, _ := git.BranchExists(filepath.Join(wsDir, "repos", "infra"), "feature/SKIP-1-web")
	if exists {
		t.Error("infra should not have branch feature/SKIP-1-web")
	}
}

func TestRunStart_remoteBranchTracking(t *testing.T) {
	wsDir := t.TempDir()
	branch := "feature/TRACK-1-test"
	bare := testutil.CreateBareRepoWithBranch(t, branch)

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
	root2.SetArgs([]string{"--root", wsDir, "start", "TRACK-1", "test", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	cur, _ := git.CurrentBranch(dir)
	if cur != branch {
		t.Errorf("expected tracking branch %s, got %s", branch, cur)
	}
}

func TestRunStart_strategySafe_dirtyRepoError(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "SAFE-1", "test", "--from", "HEAD"})
	err := root2.Execute()
	if err == nil {
		t.Fatal("expected error for dirty repo with safe strategy")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("expected error about uncommitted changes, got: %v", err)
	}
}

func TestRunStart_strategyStash_dirtyRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "STASH-1", "test", "--from", "HEAD", "--strategy", "stash"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start with stash strategy failed: %v", err)
	}

	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/STASH-1-test" {
		t.Errorf("expected branch feature/STASH-1-test, got %s", branch)
	}

	dirty, _ := git.IsDirty(dir)
	if dirty {
		t.Error("repo should not be dirty after stash")
	}
}

func TestRunStart_strategyReset_requiresForce(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "start", "RST-1", "test", "--strategy", "reset"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when using reset without --force")
	}
	if !strings.Contains(err.Error(), "--strategy reset requires --force") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunStart_strategyReset_withForce(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "RST-2", "test", "--from", "HEAD", "--strategy", "reset", "--force"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start with reset+force failed: %v", err)
	}

	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/RST-2-test" {
		t.Errorf("expected branch feature/RST-2-test, got %s", branch)
	}

	dirty, _ := git.IsDirty(dir)
	if dirty {
		t.Error("repo should not be dirty after reset")
	}
}

func TestRunStart_skipsUnclonedRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--only", "backend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "UNCL-1", "test", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	backendDir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(backendDir)
	if branch != "feature/UNCL-1-test" {
		t.Errorf("backend: expected branch feature/UNCL-1-test, got %s", branch)
	}

	frontendDir := filepath.Join(wsDir, "repos", "frontend")
	if git.IsCloned(frontendDir) {
		t.Error("frontend should not be cloned")
	}
}

func TestRunStart_dryRun_multiRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 3)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "DRY-2", "test", "--from", "HEAD", "--dry-run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start --dry-run failed: %v", err)
	}

	for _, name := range []string{"backend", "frontend", "infra"} {
		d := filepath.Join(wsDir, "repos", name)
		exists, _ := git.BranchExists(d, "feature/DRY-2-test")
		if exists {
			t.Errorf("repo %s: branch should not be created in dry-run mode", name)
		}
	}
}

func TestRunStart_multiRepo_dirtySubset(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 3)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "frontend")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "DIRTY-1", "test", "--from", "HEAD"})
	err := root2.Execute()
	if err == nil {
		t.Fatal("expected error when one repo is dirty with safe strategy")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") {
		t.Errorf("expected error about uncommitted changes, got: %v", err)
	}
}

func TestRunStart_baseRefDefaults(t *testing.T) {
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

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "BASE-1", "test"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/BASE-1-test" {
		t.Errorf("expected branch feature/BASE-1-test, got %s", branch)
	}
}

func TestRunStart_baseRefPerRepo(t *testing.T) {
	wsDir := t.TempDir()
	bare1 := testutil.CreateBareRepo(t)
	bare2 := testutil.CreateBareRepo(t)

	wsYAML := fmt.Sprintf("version: 1\nname: test\nrepos_root: repos\ndefaults:\n  base_ref: main\nrepos:\n  - id: backend\n    url: %s\n    path: repos/backend\n    ref: main\n  - id: frontend\n    url: %s\n    path: repos/frontend\n    ref: main\n    base_ref: main\n", bare1, bare2)
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "PERREPO-1", "test"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	for _, name := range []string{"backend", "frontend"} {
		dir := filepath.Join(wsDir, "repos", name)
		branch, _ := git.CurrentBranch(dir)
		if branch != "feature/PERREPO-1-test" {
			t.Errorf("repo %s: expected branch feature/PERREPO-1-test, got %s", name, branch)
		}
	}
}

func TestRunStart_fromFlagOverridesBaseRef(t *testing.T) {
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
	root2.SetArgs([]string{"--root", wsDir, "start", "FROMFLAG-1", "test", "--from", "HEAD"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/FROMFLAG-1-test" {
		t.Errorf("expected branch feature/FROMFLAG-1-test, got %s", branch)
	}
}

func TestRunStart_noUpstreamToBaseBranch(t *testing.T) {
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

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "NOTRACK-1", "test"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch, _ := git.CurrentBranch(dir)
	if branch != "feature/NOTRACK-1-test" {
		t.Errorf("expected branch feature/NOTRACK-1-test, got %s", branch)
	}

	// Verify upstream is NOT set to origin/main.
	// With --no-track, git config branch.<name>.remote should not exist.
	cmd := exec.Command("git", "config", "branch.feature/NOTRACK-1-test.remote")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err == nil {
		t.Errorf("expected no upstream remote, but got %q", strings.TrimSpace(string(out)))
	}
}

func TestRunStart_noBaseRefNoFromError(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// No --from, no base_ref in workspace.yaml → should error
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "NOBASE-1", "test"})
	err := root2.Execute()
	if err == nil {
		t.Fatal("expected error when base_ref and --from are both unset")
	}
	if !strings.Contains(err.Error(), "base_ref is not configured") {
		t.Errorf("expected error about base_ref not configured, got: %v", err)
	}
}

func TestRunStart_existingLocalBranch(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	branch := "feature/EXIST-1-test"

	if err := git.CreateBranch(dir, branch, "HEAD"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	if err := git.Checkout(dir, "main"); err != nil {
		t.Fatalf("failed to checkout main: %v", err)
	}

	cur, _ := git.CurrentBranch(dir)
	if cur != "main" {
		t.Fatalf("expected to be on main, got %s", cur)
	}

	// No --from and no base_ref → should still succeed because branch already exists.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "EXIST-1", "test"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	cur, _ = git.CurrentBranch(dir)
	if cur != branch {
		t.Errorf("expected branch %s, got %s", branch, cur)
	}
}

func TestRunStart_noBaseRef_stashNotApplied(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Make the repo dirty.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// No --from, no base_ref, --strategy stash → should error about base_ref
	// WITHOUT stashing first (no side effects before config error).
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "start", "STASH-NOBASE-1", "test", "--strategy", "stash"})
	err := root2.Execute()
	if err == nil {
		t.Fatal("expected error when base_ref and --from are both unset")
	}
	if !strings.Contains(err.Error(), "base_ref is not configured") {
		t.Errorf("expected error about base_ref not configured, got: %v", err)
	}

	// Verify repo is still dirty (stash was NOT applied).
	dirty, _ := git.IsDirty(dir)
	if !dirty {
		t.Error("repo should still be dirty — stash should not have been applied before config error")
	}
}
