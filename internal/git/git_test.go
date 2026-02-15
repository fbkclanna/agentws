package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/testutil"
)

func TestCloneAndIsCloned(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dest := filepath.Join(t.TempDir(), "cloned")

	if err := Clone(bare, dest, CloneOpts{}); err != nil {
		t.Fatalf("clone: %v", err)
	}
	if !IsCloned(dest) {
		t.Error("expected IsCloned to be true after clone")
	}
}

func TestClone_withDepth(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dest := filepath.Join(t.TempDir(), "shallow")
	depth := 1

	if err := Clone(bare, dest, CloneOpts{Depth: &depth}); err != nil {
		t.Fatalf("clone with depth: %v", err)
	}
	if !IsCloned(dest) {
		t.Error("expected cloned")
	}
}

func TestCurrentBranch(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dest := filepath.Join(t.TempDir(), "repo")
	if err := Clone(bare, dest, CloneOpts{}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	branch, err := CurrentBranch(dest)
	if err != nil {
		t.Fatal(err)
	}
	// Default branch from our test bare repo should be main or master.
	if branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestHeadCommit(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dest := filepath.Join(t.TempDir(), "repo")
	if err := Clone(bare, dest, CloneOpts{}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	sha, err := HeadCommit(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) < 7 {
		t.Errorf("short sha too short: %q", sha)
	}
}

func TestIsDirty(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dest := filepath.Join(t.TempDir(), "repo")
	if err := Clone(bare, dest, CloneOpts{}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	dirty, err := IsDirty(dest)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Error("expected clean repo after fresh clone")
	}

	// Make it dirty.
	if err := os.WriteFile(filepath.Join(dest, "dirty.txt"), []byte("x"), 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	dirty, err = IsDirty(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Error("expected dirty after creating untracked file")
	}
}

func TestBranchExists(t *testing.T) {
	bare := testutil.CreateBareRepoWithBranch(t, "feature/test")
	dest := filepath.Join(t.TempDir(), "repo")
	if err := Clone(bare, dest, CloneOpts{}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	// Local branch should not exist yet.
	exists, err := BranchExists(dest, "feature/test")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("expected local branch to not exist before checkout")
	}

	// Remote branch should exist.
	remoteExists, err := RemoteBranchExists(dest, "feature/test")
	if err != nil {
		t.Fatal(err)
	}
	if !remoteExists {
		t.Error("expected remote branch to exist")
	}
}

func TestCreateBranch(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dest := filepath.Join(t.TempDir(), "repo")
	if err := Clone(bare, dest, CloneOpts{}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	if err := CreateBranch(dest, "new-branch", "HEAD"); err != nil {
		t.Fatal(err)
	}

	branch, _ := CurrentBranch(dest)
	if branch != "new-branch" {
		t.Errorf("expected to be on new-branch, got %s", branch)
	}
}

func TestStash(t *testing.T) {
	bare := testutil.CreateBareRepo(t)
	dest := filepath.Join(t.TempDir(), "repo")
	if err := Clone(bare, dest, CloneOpts{}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	// Create a tracked file change to stash.
	if err := os.WriteFile(filepath.Join(dest, "README.md"), []byte("modified\n"), 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	if err := Stash(dest); err != nil {
		t.Fatal(err)
	}

	dirty, _ := IsDirty(dest)
	if dirty {
		t.Error("expected clean after stash")
	}
}

func TestDefaultBranch(t *testing.T) {
	bare := testutil.CreateBareRepo(t)

	branch, err := DefaultBranch(bare)
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}
	if branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestDefaultBranch_error(t *testing.T) {
	// Non-existent URL should return an error.
	_, err := DefaultBranch("/nonexistent/repo.git")
	if err == nil {
		t.Fatal("expected error for non-existent repo")
	}
}

func TestIsCloned_notCloned(t *testing.T) {
	if IsCloned("/nonexistent/path") {
		t.Error("expected false for nonexistent path")
	}
}
