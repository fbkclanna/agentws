package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/fbkclanna/agentws/internal/testutil"
	"gopkg.in/yaml.v3"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  Strategy
		err   bool
	}{
		{"safe", StrategySafe, false},
		{"stash", StrategyStash, false},
		{"reset", StrategyReset, false},
		{"", StrategySafe, false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseStrategy(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("ParseStrategy(%q) error = %v, wantErr %v", tt.input, err, tt.err)
			}
			if got != tt.want {
				t.Errorf("ParseStrategy(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// writeManifest is a test helper that writes a workspace.yaml to the given dir.
func writeManifest(t *testing.T, dir string, ws *manifest.Workspace) {
	t.Helper()
	data, err := yaml.Marshal(ws)
	if err != nil {
		t.Fatalf("marshaling manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), data, 0600); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
}

func TestLoad(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dir := t.TempDir()

	ws := &manifest.Workspace{
		Version:   1,
		Name:      "test-ws",
		ReposRoot: "repos",
		Repos: []manifest.Repo{
			{ID: "backend", URL: bare, Path: "repos/backend", Ref: "main"},
		},
	}
	writeManifest(t, dir, ws)

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if ctx.Manifest.Name != "test-ws" {
		t.Errorf("Manifest.Name = %q, want %q", ctx.Manifest.Name, "test-ws")
	}
	if ctx.Lock != nil {
		t.Error("Lock should be nil when no lock file exists")
	}
	if ctx.ManifestPath != filepath.Join(ctx.Root, "workspace.yaml") {
		t.Errorf("ManifestPath = %q, unexpected", ctx.ManifestPath)
	}
	if ctx.LockPath != filepath.Join(ctx.Root, "workspace.lock.yaml") {
		t.Errorf("LockPath = %q, unexpected", ctx.LockPath)
	}
}

func TestLoad_withLock(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dir := t.TempDir()

	ws := &manifest.Workspace{
		Version:   1,
		Name:      "test-ws",
		ReposRoot: "repos",
		Repos: []manifest.Repo{
			{ID: "backend", URL: bare, Path: "repos/backend", Ref: "main"},
		},
	}
	writeManifest(t, dir, ws)

	// Write a lock file.
	lockData := []byte(`version: 1
name: test-ws
generated_at: "2026-02-15T00:00:00Z"
tool_version: "0.1.0"
repos:
  backend:
    url: ` + bare + `
    ref: main
    commit: "abc1234"
`)
	if err := os.WriteFile(filepath.Join(dir, "workspace.lock.yaml"), lockData, 0600); err != nil {
		t.Fatal(err)
	}

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if ctx.Lock == nil {
		t.Fatal("Lock should not be nil when lock file exists")
	}
	if ctx.Lock.Name != "test-ws" {
		t.Errorf("Lock.Name = %q, want %q", ctx.Lock.Name, "test-ws")
	}
	repo, ok := ctx.Lock.Repos["backend"]
	if !ok {
		t.Fatal("Lock should contain backend repo")
	}
	if repo.Commit != "abc1234" {
		t.Errorf("Lock.Repos[backend].Commit = %q, want %q", repo.Commit, "abc1234")
	}
}

func TestLoad_missingManifest(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() should fail when workspace.yaml is missing")
	}
}

func TestLoad_invalidManifest(t *testing.T) {
	dir := t.TempDir()

	// Write invalid YAML.
	if err := os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte(":::invalid"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() should fail with invalid YAML")
	}
}

func TestRepoDir(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dir := t.TempDir()

	ws := &manifest.Workspace{
		Version:   1,
		Name:      "test-ws",
		ReposRoot: "repos",
		Repos: []manifest.Repo{
			{ID: "backend", URL: bare, Path: "repos/backend", Ref: "main"},
		},
	}
	writeManifest(t, dir, ws)

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	repo := ctx.Manifest.Repos[0]
	got := ctx.RepoDir(repo)
	want := filepath.Join(ctx.Root, "repos/backend")
	if got != want {
		t.Errorf("RepoDir() = %q, want %q", got, want)
	}
}
