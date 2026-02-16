package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/testutil"
)

// --- Unit Tests ---

func TestBuildNewRepos_singleURL(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	repos, err := buildNewRepos([]string{bare}, "repos", "", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	r := repos[0]
	if r.URL != bare {
		t.Errorf("url = %q, want %q", r.URL, bare)
	}
	if r.ID == "" {
		t.Error("id should be inferred from URL")
	}
	if r.Path == "" {
		t.Error("path should be set")
	}
	if r.Ref == "" {
		t.Error("ref should be detected or defaulted")
	}
}

func TestBuildNewRepos_idOverride(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	repos, err := buildNewRepos([]string{bare}, "repos", "custom-id", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repos[0].ID != "custom-id" {
		t.Errorf("id = %q, want %q", repos[0].ID, "custom-id")
	}
	if repos[0].Path != "repos/custom-id" {
		t.Errorf("path = %q, want %q", repos[0].Path, "repos/custom-id")
	}
}

func TestBuildNewRepos_pathOverride(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	repos, err := buildNewRepos([]string{bare}, "repos", "", "custom/dir", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repos[0].Path != "custom/dir" {
		t.Errorf("path = %q, want %q", repos[0].Path, "custom/dir")
	}
}

func TestBuildNewRepos_emptyURL(t *testing.T) {
	_, err := buildNewRepos([]string{""}, "repos", "", "", "", nil)
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestFindConflicts_duplicateID(t *testing.T) {
	existing := []manifest.Repo{{ID: "backend", URL: "a", Path: "repos/backend"}}
	newRepos := []manifest.Repo{{ID: "backend", URL: "b", Path: "repos/other"}}
	if err := findConflicts(existing, newRepos); err == nil {
		t.Fatal("expected error for duplicate ID")
	}
}

func TestFindConflicts_duplicatePath(t *testing.T) {
	existing := []manifest.Repo{{ID: "backend", URL: "a", Path: "repos/backend"}}
	newRepos := []manifest.Repo{{ID: "other", URL: "b", Path: "repos/backend"}}
	if err := findConflicts(existing, newRepos); err == nil {
		t.Fatal("expected error for duplicate path")
	}
}

func TestFindConflicts_noConflict(t *testing.T) {
	existing := []manifest.Repo{{ID: "backend", URL: "a", Path: "repos/backend"}}
	newRepos := []manifest.Repo{{ID: "frontend", URL: "b", Path: "repos/frontend"}}
	if err := findConflicts(existing, newRepos); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- E2E Tests ---

func TestRunAdd_singleURL(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if len(ws.Repos) != 2 {
		t.Fatalf("repos count = %d, want 2", len(ws.Repos))
	}
	added := ws.Repos[1]
	if added.URL != bare {
		t.Errorf("url = %q, want %q", added.URL, bare)
	}
}

func TestRunAdd_multipleURLs(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare1 := testutil.CreateBareRepo(t)
	bare2 := testutil.CreateBareRepo(t)

	// bare repos have the same basename "repo.git", so we need unique IDs.
	// Use --ref to avoid ls-remote but let each get a unique path by bare dir name.
	// Actually, both will have id="repo" which will conflict. Use separate invocations or rename.
	// Better approach: use two separate calls or leverage the fact that CreateBareRepo gives unique dirs.
	// The ID will be "repo" for both. We need to add them one at a time with --id.

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare1, "--id", "repo1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add repo1 failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "add", bare2, "--id", "repo2"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("add repo2 failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if len(ws.Repos) != 2 {
		t.Fatalf("repos count = %d, want 2", len(ws.Repos))
	}
}

func TestRunAdd_duplicateIDError(t *testing.T) {
	wsDir, bareRepos := setupWorkspace(t, 1)

	// Try to add with the same ID as existing repo.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bareRepos[0], "--id", "backend"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}

	// Manifest should be unchanged.
	ws, loadErr := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if loadErr != nil {
		t.Fatalf("load manifest: %v", loadErr)
	}
	if len(ws.Repos) != 1 {
		t.Errorf("repos count = %d, want 1 (unchanged)", len(ws.Repos))
	}
}

func TestRunAdd_idOverride(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--id", "custom"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if ws.Repos[0].ID != "custom" {
		t.Errorf("id = %q, want %q", ws.Repos[0].ID, "custom")
	}
}

func TestRunAdd_pathOverride(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--path", "custom/dir"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if ws.Repos[0].Path != "custom/dir" {
		t.Errorf("path = %q, want %q", ws.Repos[0].Path, "custom/dir")
	}
}

func TestRunAdd_refOverride(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--ref", "develop"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if ws.Repos[0].Ref != "develop" {
		t.Errorf("ref = %q, want %q", ws.Repos[0].Ref, "develop")
	}
}

func TestRunAdd_tagFlag(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--tag", "core", "--tag", "backend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	tags := ws.Repos[0].Tags
	if len(tags) != 2 || tags[0] != "core" || tags[1] != "backend" {
		t.Errorf("tags = %v, want [core backend]", tags)
	}
}

func TestRunAdd_idWithMultipleURLsError(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare1 := testutil.CreateBareRepo(t)
	bare2 := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare1, bare2, "--id", "x"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --id used with multiple URLs")
	}
}

func TestRunAdd_withSync(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--sync", "--id", "synced"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add --sync failed: %v", err)
	}

	cloneDir := filepath.Join(wsDir, "repos", "synced")
	if !git.IsCloned(cloneDir) {
		t.Error("repo should be cloned after --sync")
	}
}

func TestRunAdd_syncSkipsAlreadyCloned(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	// Pre-clone.
	cloneDir := filepath.Join(wsDir, "repos", "precloned")
	if err := git.Clone(bare, cloneDir, git.CloneOpts{}); err != nil {
		t.Fatalf("pre-clone failed: %v", err)
	}

	root := newRootCmd()
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--sync", "--id", "precloned", "--path", "repos/precloned"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add --sync failed: %v", err)
	}

	if !bytes.Contains(stderr.Bytes(), []byte("Skipping")) {
		t.Error("expected skip message for already cloned repo")
	}
}

func TestRunAdd_noWorkspaceError(t *testing.T) {
	dir := t.TempDir() // No workspace.yaml.
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "add", bare})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no workspace.yaml exists")
	}
}

func TestRunAdd_jsonOutput(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--json", "--id", "jsontest"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add --json failed: %v", err)
	}

	var repos []manifest.Repo
	if err := json.Unmarshal(stdout.Bytes(), &repos); err != nil {
		t.Fatalf("JSON parse failed: %v\noutput: %s", err, stdout.String())
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos in JSON, want 1", len(repos))
	}
	if repos[0].ID != "jsontest" {
		t.Errorf("id = %q, want %q", repos[0].ID, "jsontest")
	}
}

func TestRunAdd_preservesExistingRepos(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--id", "newrepo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if len(ws.Repos) != 3 {
		t.Fatalf("repos count = %d, want 3", len(ws.Repos))
	}
	// Verify existing repos are preserved.
	if ws.Repos[0].ID != "backend" {
		t.Errorf("first repo = %q, want %q", ws.Repos[0].ID, "backend")
	}
	if ws.Repos[1].ID != "frontend" {
		t.Errorf("second repo = %q, want %q", ws.Repos[1].ID, "frontend")
	}
	if ws.Repos[2].ID != "newrepo" {
		t.Errorf("third repo = %q, want %q", ws.Repos[2].ID, "newrepo")
	}
}

func TestRunAdd_addThenStatus(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	// Add repo.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--id", "statustest"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	// Run status.
	root2 := newRootCmd()
	var stdout bytes.Buffer
	root2.SetOut(&stdout)
	root2.SetArgs([]string{"--root", wsDir, "status", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("status failed: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("statustest")) {
		t.Error("status output should contain added repo")
	}
}

func TestRunAdd_addThenSync(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 0)
	bare := testutil.CreateBareRepo(t)

	// Add without sync.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--id", "synctest"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	cloneDir := filepath.Join(wsDir, "repos", "synctest")
	if git.IsCloned(cloneDir) {
		t.Fatal("repo should NOT be cloned yet")
	}

	// Run sync.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if !git.IsCloned(cloneDir) {
		t.Error("repo should be cloned after sync")
	}
}

func TestRunAdd_duplicatePathError(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1) // Has "backend" at "repos/backend".
	bare := testutil.CreateBareRepo(t)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--id", "other", "--path", filepath.Join("repos", "backend")})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate path")
	}

	// Manifest should be unchanged.
	ws, loadErr := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if loadErr != nil {
		t.Fatalf("load manifest: %v", loadErr)
	}
	if len(ws.Repos) != 1 {
		t.Errorf("repos count = %d, want 1 (unchanged)", len(ws.Repos))
	}
}

func TestRunAdd_emptyReposRoot(t *testing.T) {
	// Create workspace with empty repos_root.
	wsDir := t.TempDir()
	bare := testutil.CreateBareRepo(t)

	data := []byte(`version: 1
name: test
repos: []
`)
	if err := os.WriteFile(filepath.Join(wsDir, "workspace.yaml"), data, 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "add", bare, "--id", "myrepo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(wsDir, "workspace.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	// With empty repos_root, path should just be the id.
	if ws.Repos[0].Path != "myrepo" {
		t.Errorf("path = %q, want %q", ws.Repos[0].Path, "myrepo")
	}
}
