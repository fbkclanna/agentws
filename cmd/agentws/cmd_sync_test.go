package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/lock"
	"github.com/fbkclanna/agentws/internal/testutil"
	"gopkg.in/yaml.v3"
)

// setupWorkspace creates a temp workspace with a manifest pointing to test bare repos.
func setupWorkspace(t *testing.T, repoCount int) (wsDir string, bareRepos []string) {
	t.Helper()
	wsDir = t.TempDir()

	type repo struct {
		ID   string `yaml:"id"`
		URL  string `yaml:"url"`
		Path string `yaml:"path"`
		Ref  string `yaml:"ref"`
	}
	type ws struct {
		Version   int    `yaml:"version"`
		Name      string `yaml:"name"`
		ReposRoot string `yaml:"repos_root"`
		Repos     []repo `yaml:"repos"`
	}

	m := ws{
		Version:   1,
		Name:      "test",
		ReposRoot: "repos",
	}

	for i := range repoCount {
		bare := testutil.CreateBareRepo(t)
		bareRepos = append(bareRepos, bare)
		m.Repos = append(m.Repos, repo{
			ID:   repoName(i),
			URL:  bare,
			Path: filepath.Join("repos", repoName(i)),
			Ref:  "main",
		})
	}

	data, _ := yaml.Marshal(&m)
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return wsDir, bareRepos
}

func repoName(i int) string {
	names := []string{"backend", "frontend", "infra", "analytics"}
	if i < len(names) {
		return names[i]
	}
	return "repo" + string(rune('a'+i))
}

func TestRunSync_clonesRepos(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Both repos should be cloned.
	for _, name := range []string{"backend", "frontend"} {
		dir := filepath.Join(wsDir, "repos", name)
		if !git.IsCloned(dir) {
			t.Errorf("repo %s should be cloned", name)
		}
	}
}

func TestRunSync_idempotent(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// Run sync twice.
	for i := range 2 {
		root := newRootCmd()
		root.SetArgs([]string{"--root", wsDir, "sync"})
		if err := root.Execute(); err != nil {
			t.Fatalf("sync #%d failed: %v", i+1, err)
		}
	}

	dir := filepath.Join(wsDir, "repos", "backend")
	if !git.IsCloned(dir) {
		t.Error("repo should be cloned after two syncs")
	}
}

func TestRunSync_onlyFilter(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--only", "backend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --only failed: %v", err)
	}

	if !git.IsCloned(filepath.Join(wsDir, "repos", "backend")) {
		t.Error("backend should be cloned")
	}
	if git.IsCloned(filepath.Join(wsDir, "repos", "frontend")) {
		t.Error("frontend should NOT be cloned with --only backend")
	}
}

func TestRunSync_skipFilter(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--skip", "frontend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --skip failed: %v", err)
	}

	if !git.IsCloned(filepath.Join(wsDir, "repos", "backend")) {
		t.Error("backend should be cloned")
	}
	if git.IsCloned(filepath.Join(wsDir, "repos", "frontend")) {
		t.Error("frontend should NOT be cloned with --skip")
	}
}

func TestRunSync_updateLock(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--update-lock"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --update-lock failed: %v", err)
	}

	lockPath := filepath.Join(wsDir, "workspace.lock.yaml")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file not created: %v", err)
	}
}

func TestRunSync_strategySafe_skipsDirty(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// First sync to clone.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Make dirty.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// Sync again with safe strategy — should not error.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "sync", "--strategy", "safe"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("sync with safe strategy failed: %v", err)
	}
}

func TestRunSync_resetRequiresForce(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--strategy", "reset"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when using reset without --force")
	}
}

func TestRunSync_strategyStash(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// Initial sync to clone.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Make dirty by modifying a tracked file (git stash only stashes tracked changes).
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Sync with stash strategy.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "sync", "--strategy", "stash"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("sync --strategy stash failed: %v", err)
	}

	// Repo should no longer be dirty after stash.
	dirty, err := git.IsDirty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Error("repo should not be dirty after stash sync")
	}
}

func TestRunSync_strategyResetWithForce(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// Initial sync.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Make dirty by modifying a tracked file.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Sync with reset + force.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "sync", "--strategy", "reset", "--force"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("sync --strategy reset --force failed: %v", err)
	}

	dirty, err := git.IsDirty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Error("repo should not be dirty after reset sync")
	}
}

func TestRunSync_lockCheckout(t *testing.T) {
	wsDir, bareRepos := setupWorkspace(t, 1)

	// Initial sync.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Pin current HEAD.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "pin"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("pin failed: %v", err)
	}

	// Read the pinned commit.
	dir := filepath.Join(wsDir, "repos", "backend")
	pinnedCommit, err := git.HeadCommitFull(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Advance the bare repo.
	testutil.PushNewCommit(t, bareRepos[0])

	// Sync with --lock should checkout the pinned commit.
	root3 := newRootCmd()
	root3.SetArgs([]string{"--root", wsDir, "sync", "--lock"})
	if err := root3.Execute(); err != nil {
		t.Fatalf("sync --lock failed: %v", err)
	}

	currentCommit, err := git.HeadCommitFull(dir)
	if err != nil {
		t.Fatal(err)
	}
	if currentCommit != pinnedCommit {
		t.Errorf("expected HEAD=%s (pinned), got %s", pinnedCommit[:7], currentCommit[:7])
	}
}

func TestRunSync_lockWithoutLockFile(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--lock"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --lock without lock file")
	}
	if !strings.Contains(err.Error(), "no workspace.lock.yaml found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSync_profileFilter(t *testing.T) {
	wsDir := t.TempDir()
	bare1 := testutil.CreateBareRepo(t)
	bare2 := testutil.CreateBareRepo(t)

	wsYAML := fmt.Sprintf(`version: 1
name: test
repos_root: repos
profiles:
  backend-only:
    include_tags: [backend]
repos:
  - id: api
    url: %s
    path: repos/api
    ref: main
    tags: [backend]
  - id: web
    url: %s
    path: repos/web
    ref: main
    tags: [frontend]
`, bare1, bare2)

	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--profile", "backend-only"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --profile failed: %v", err)
	}

	if !git.IsCloned(filepath.Join(wsDir, "repos", "api")) {
		t.Error("api should be cloned (matches backend tag)")
	}
	if git.IsCloned(filepath.Join(wsDir, "repos", "web")) {
		t.Error("web should NOT be cloned (frontend tag)")
	}
}

func TestRunSync_postSync(t *testing.T) {
	wsDir := t.TempDir()
	bare := testutil.CreateBareRepo(t)

	wsYAML := fmt.Sprintf(`version: 1
name: test
repos_root: repos
repos:
  - id: backend
    url: %s
    path: repos/backend
    ref: main
    post_sync:
      - name: create marker
        cmd: ["touch", "marker.txt"]
`, bare)

	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync with post_sync failed: %v", err)
	}

	markerPath := filepath.Join(wsDir, "repos", "backend", "marker.txt")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("marker.txt not created by post_sync: %v", err)
	}
}

func TestRunSync_fetchOnAlreadyCloned(t *testing.T) {
	wsDir, bareRepos := setupWorkspace(t, 1)

	// Initial sync.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Advance bare repo.
	testutil.PushNewCommit(t, bareRepos[0])

	// Re-sync should complete without error (fetch succeeds on existing clone).
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("re-sync failed: %v", err)
	}

	// Verify repo is still cloned and in valid state.
	dir := filepath.Join(wsDir, "repos", "backend")
	if !git.IsCloned(dir) {
		t.Error("repo should still be cloned after re-sync")
	}
}

// --- Local repo sync tests ---

func TestRunSync_localRepo_init(t *testing.T) {
	wsDir := t.TempDir()
	wsYAML := `version: 1
name: test
repos_root: repos
repos:
  - id: local-svc
    local: true
    path: repos/local-svc
    ref: main
`
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	dir := filepath.Join(wsDir, "repos", "local-svc")
	if !git.IsCloned(dir) {
		t.Error("local repo should be initialized by sync")
	}
	if git.HasRemote(dir) {
		t.Error("local repo should not have a remote")
	}
}

func TestRunSync_localRepo_skipsFetch(t *testing.T) {
	wsDir := t.TempDir()
	wsYAML := `version: 1
name: test
repos_root: repos
repos:
  - id: local-svc
    local: true
    path: repos/local-svc
    ref: main
`
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// First sync to init.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Second sync should not fail (no fetch for local repos).
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("re-sync of local repo failed: %v", err)
	}
}

func TestRunSync_localRepo_mixedRepos(t *testing.T) {
	wsDir := t.TempDir()
	bare := testutil.CreateBareRepo(t)

	wsYAML := fmt.Sprintf(`version: 1
name: test
repos_root: repos
repos:
  - id: remote-svc
    url: %s
    path: repos/remote-svc
    ref: main
  - id: local-svc
    local: true
    path: repos/local-svc
    ref: main
`, bare)

	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), []byte(wsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync mixed repos failed: %v", err)
	}

	// Both should be initialized/cloned.
	if !git.IsCloned(filepath.Join(wsDir, "repos", "remote-svc")) {
		t.Error("remote repo should be cloned")
	}
	if !git.IsCloned(filepath.Join(wsDir, "repos", "local-svc")) {
		t.Error("local repo should be initialized")
	}
}

func TestRunSync_lockCheckoutVerifyLock(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// Sync and pin.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--update-lock"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --update-lock failed: %v", err)
	}

	lockPath := filepath.Join(wsDir, "workspace.lock.yaml")
	lf, err := lock.Load(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(lf.Repos) != 1 {
		t.Fatalf("expected 1 repo in lock, got %d", len(lf.Repos))
	}
	if lf.Repos["backend"] == nil {
		t.Fatal("backend not in lock file")
	}
	if lf.Repos["backend"].Commit == "" {
		t.Error("backend commit should not be empty in lock file")
	}
}
