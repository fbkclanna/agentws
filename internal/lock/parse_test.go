package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_valid(t *testing.T) {
	data := []byte(`
version: 1
name: foo
generated_at: "2026-02-15T12:34:56+09:00"
tool_version: "0.1.0"
repos:
  backend:
    url: git@github.com:org/foo-backend.git
    ref: main
    commit: "a1b2c3d4e5f6"
  analytics:
    url: git@github.com:org/foo-analytics.git
    ref: main
    commit: "deadbeef1234"
`)
	lf, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lf.Version != 1 {
		t.Errorf("version = %d, want 1", lf.Version)
	}
	if lf.Name != "foo" {
		t.Errorf("name = %q, want %q", lf.Name, "foo")
	}
	if len(lf.Repos) != 2 {
		t.Errorf("repos count = %d, want 2", len(lf.Repos))
	}
	be := lf.Repos["backend"]
	if be == nil {
		t.Fatal("backend repo not found")
	}
	if be.Commit != "a1b2c3d4e5f6" {
		t.Errorf("commit = %q", be.Commit)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace.lock.yaml")

	lf := &File{
		Version:     1,
		Name:        "test",
		GeneratedAt: "2026-01-01T00:00:00Z",
		ToolVersion: "dev",
		Repos: map[string]*Repo{
			"svc": {
				URL:    "git@github.com:org/svc.git",
				Ref:    "main",
				Commit: "abc123",
			},
		},
	}

	if err := Save(path, lf); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatal("file should exist after save")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Name != "test" {
		t.Errorf("name = %q, want %q", loaded.Name, "test")
	}
	if loaded.Repos["svc"].Commit != "abc123" {
		t.Errorf("commit = %q, want %q", loaded.Repos["svc"].Commit, "abc123")
	}
}
