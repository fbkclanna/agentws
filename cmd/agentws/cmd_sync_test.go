package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
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
	return "repo" + string(rune('a'+i)) //nolint:gosec // i is always small
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
