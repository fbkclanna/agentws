package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/manifest"
)

func TestRunInit_fromLocalFile(t *testing.T) {
	dir := t.TempDir()

	// Create a source manifest.
	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: imported
repos_root: repos
repos:
  - id: svc
    url: git@github.com:org/svc.git
    path: repos/svc
`)
	if err := os.WriteFile(src, data, 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "imported", "--from", src})

	if err := root.Execute(); err != nil {
		t.Fatalf("init --from failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(dir, "imported", "workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "imported" {
		t.Errorf("name = %q, want %q", ws.Name, "imported")
	}
	if len(ws.Repos) != 1 || ws.Repos[0].ID != "svc" {
		t.Errorf("unexpected repos: %+v", ws.Repos)
	}
}

func TestRunInit_alreadyExists(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "myws")
	if err := os.MkdirAll(wsDir, 0755); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "myws"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when workspace already exists")
	}
}

func TestRunInit_force(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "myws")
	if err := os.MkdirAll(wsDir, 0755); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}

	// Use --from to avoid interactive mode.
	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: myws
repos_root: repos
repos:
  - id: app
    url: git@github.com:org/app.git
    path: repos/app
`)
	if err := os.WriteFile(src, data, 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "myws", "--force", "--from", src})

	if err := root.Execute(); err != nil {
		t.Fatalf("init --force failed: %v", err)
	}

	manifestPath := filepath.Join(wsDir, "workspace.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest not created with --force: %v", err)
	}
}
