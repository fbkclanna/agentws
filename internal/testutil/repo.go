package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// CreateBareRepo creates a bare git repository with an initial commit in a temp directory.
// Returns the path to the bare repo.
func CreateBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bare := filepath.Join(dir, "repo.git")

	// Create a working repo first, then clone it bare.
	work := filepath.Join(dir, "work")
	run(t, dir, "git", "init", "-b", "main", work)
	run(t, work, "git", "config", "user.email", "test@example.com")
	run(t, work, "git", "config", "user.name", "Test")

	// Create an initial commit.
	readme := filepath.Join(work, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "-m", "initial commit")

	// Clone as bare.
	run(t, dir, "git", "clone", "--bare", work, bare)
	return bare
}

// CreateBareRepoWithBranch creates a bare repo with a given branch.
func CreateBareRepoWithBranch(t *testing.T, branch string) string {
	t.Helper()
	dir := t.TempDir()
	bare := filepath.Join(dir, "repo.git")

	work := filepath.Join(dir, "work")
	run(t, dir, "git", "init", "-b", "main", work)
	run(t, work, "git", "config", "user.email", "test@example.com")
	run(t, work, "git", "config", "user.name", "Test")

	readme := filepath.Join(work, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "-m", "initial commit")
	run(t, work, "git", "checkout", "-b", branch)

	f := filepath.Join(work, "feature.txt")
	if err := os.WriteFile(f, []byte("feature\n"), 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "-m", "feature commit")

	// Switch back to main so the bare repo's HEAD points to main.
	run(t, work, "git", "checkout", "main")

	run(t, dir, "git", "clone", "--bare", work, bare)
	return bare
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %s %v failed: %v", name, args, err)
	}
}
