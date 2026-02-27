package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
)

func TestInitLocalRepo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-service")

	if err := initLocalRepo(dir); err != nil {
		t.Fatalf("initLocalRepo failed: %v", err)
	}

	// Should be a git repo.
	if !git.IsCloned(dir) {
		t.Error("expected directory to be a git repo")
	}

	// Should have README.md.
	readmePath := filepath.Join(dir, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}
	if string(data) != "# my-service\n" {
		t.Errorf("README.md content = %q, want %q", string(data), "# my-service\n")
	}

	// Should have an initial commit (HEAD should resolve).
	commit, err := git.HeadCommit(dir)
	if err != nil {
		t.Fatalf("HeadCommit failed: %v", err)
	}
	if commit == "" {
		t.Error("expected non-empty HEAD commit")
	}

	// Should not be dirty after initial commit.
	dirty, err := git.IsDirty(dir)
	if err != nil {
		t.Fatalf("IsDirty failed: %v", err)
	}
	if dirty {
		t.Error("expected clean working tree after init")
	}

	// Should not have a remote.
	if git.HasRemote(dir) {
		t.Error("local repo should not have a remote")
	}
}

func TestInitLocalRepo_alreadyExists(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Place a file to verify it's not destroyed.
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := initLocalRepo(dir); err != nil {
		t.Fatalf("initLocalRepo on existing dir failed: %v", err)
	}

	// Original file should still exist.
	if _, err := os.Stat(filepath.Join(dir, "existing.txt")); err != nil {
		t.Error("existing.txt should still exist")
	}
}
