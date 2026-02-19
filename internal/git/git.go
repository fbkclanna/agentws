package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneOpts configures a git clone operation.
type CloneOpts struct {
	Depth        *int
	PartialClone bool
	Sparse       []string
}

// Clone clones a repository to dest with the given options.
func Clone(url, dest string, opts CloneOpts) error {
	args := []string{"clone"}

	if opts.Depth != nil && *opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", *opts.Depth))
	}
	if opts.PartialClone {
		args = append(args, "--filter=blob:none")
	}
	if len(opts.Sparse) > 0 {
		args = append(args, "--no-checkout")
	}

	args = append(args, url, dest)

	if err := run(".", args...); err != nil {
		return fmt.Errorf("cloning %s: %w", url, err)
	}

	if len(opts.Sparse) > 0 {
		if err := SparseCheckoutSet(dest, opts.Sparse); err != nil {
			return err
		}
		if err := run(dest, "checkout"); err != nil {
			return fmt.Errorf("checkout after sparse setup: %w", err)
		}
	}

	return nil
}

// Fetch runs git fetch in the given repo directory.
func Fetch(repoDir string) error {
	return run(repoDir, "fetch", "--prune")
}

// Checkout checks out the given ref.
func Checkout(repoDir, ref string) error {
	return run(repoDir, "checkout", ref)
}

// CurrentBranch returns the current branch name, or empty string if detached.
func CurrentBranch(repoDir string) (string, error) {
	out, err := output(repoDir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		// Detached HEAD: symbolic-ref fails.
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// HeadCommit returns the short SHA of HEAD.
func HeadCommit(repoDir string) (string, error) {
	out, err := output(repoDir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HeadCommitFull returns the full SHA of HEAD.
func HeadCommitFull(repoDir string) (string, error) {
	out, err := output(repoDir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// IsDirty returns true if the working tree has uncommitted changes.
func IsDirty(repoDir string) (bool, error) {
	out, err := output(repoDir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// BranchExists checks if a local branch exists.
func BranchExists(repoDir, branch string) (bool, error) {
	err := run(repoDir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	if err != nil {
		if isExitError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RemoteBranchExists checks if a remote branch exists (after fetch).
func RemoteBranchExists(repoDir, branch string) (bool, error) {
	err := run(repoDir, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
	if err != nil {
		if isExitError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateBranch creates a new branch from the given ref.
// --no-track prevents automatic upstream tracking when 'from' is a remote
// tracking ref (e.g. origin/main), which would cause git push to target the
// base branch instead of the new branch.
func CreateBranch(repoDir, branch, from string) error {
	return run(repoDir, "checkout", "-b", branch, "--no-track", from)
}

// CreateTrackingBranch creates a local tracking branch for origin/<branch>.
func CreateTrackingBranch(repoDir, branch string) error {
	return run(repoDir, "checkout", "-b", branch, "--track", "origin/"+branch)
}

// Stash stashes uncommitted changes.
func Stash(repoDir string) error {
	return run(repoDir, "stash")
}

// ResetHard resets the working tree to the given ref.
func ResetHard(repoDir, ref string) error {
	return run(repoDir, "reset", "--hard", ref)
}

// SparseCheckoutSet configures sparse-checkout with the given paths.
func SparseCheckoutSet(repoDir string, paths []string) error {
	args := append([]string{"sparse-checkout", "set", "--"}, paths...)
	return run(repoDir, args...)
}

// DefaultBranch detects the default branch of a remote repository using
// git ls-remote --symref. Returns an error if the branch cannot be detected.
func DefaultBranch(url string) (string, error) {
	out, err := output(".", "ls-remote", "--symref", url, "HEAD")
	if err != nil {
		return "", fmt.Errorf("ls-remote %s: %w", url, err)
	}
	// Expected output line: "ref: refs/heads/main\tHEAD"
	// strings.Fields splits to: ["ref:", "refs/heads/main", "HEAD"]
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[0] == "ref:" && strings.HasPrefix(parts[1], "refs/heads/") {
			return strings.TrimPrefix(parts[1], "refs/heads/"), nil
		}
	}
	return "", fmt.Errorf("default branch not found for %s", url)
}

// IsCloned returns true if the directory is a git repository.
func IsCloned(repoDir string) bool {
	info, err := os.Stat(filepath.Join(repoDir, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// run executes a git command in the given directory.
func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// output executes a git command and returns its stdout.
func output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}

// IsGitInstalled returns true if git is available on the system PATH.
func IsGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Init runs git init in the given directory.
func Init(dir string) error {
	return runQuiet(dir, "init")
}

// Add stages the given paths in the repository.
func Add(dir string, paths ...string) error {
	args := append([]string{"add", "--"}, paths...)
	return runQuiet(dir, args...)
}

// Commit creates a commit with the given message.
// If user.name or user.email is not configured globally, it sets repo-local fallback values.
func Commit(dir, message string) error {
	if err := ensureCommitIdentity(dir); err != nil {
		return fmt.Errorf("setting commit identity: %w", err)
	}
	return runQuiet(dir, "commit", "-m", message)
}

// ensureCommitIdentity sets repo-local user.name/user.email if they are not configured.
func ensureCommitIdentity(dir string) error {
	if _, err := outputQuiet(dir, "config", "user.name"); err != nil {
		if err2 := runQuiet(dir, "config", "user.name", "agentws"); err2 != nil {
			return err2
		}
	}
	if _, err := outputQuiet(dir, "config", "user.email"); err != nil {
		if err2 := runQuiet(dir, "config", "user.email", "agentws@localhost"); err2 != nil {
			return err2
		}
	}
	return nil
}

// runQuiet executes a git command without printing stdout.
// Stderr is captured and included in the error message on failure.
func runQuiet(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return nil
}

// outputQuiet executes a git command and returns its stdout without printing to the console.
func outputQuiet(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

func isExitError(err error) bool {
	_, ok := err.(*exec.ExitError)
	return ok
}
