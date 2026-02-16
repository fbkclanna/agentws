package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fbkclanna/agentws/internal/git"
	"github.com/fbkclanna/agentws/internal/manifest"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Create a new workspace interactively or from a manifest",
		Args:  cobra.ExactArgs(1),
		RunE:  runInit,
	}
	cmd.Flags().String("from", "", "Import manifest from local path or repo#path")
	cmd.Flags().Bool("force", false, "Overwrite existing workspace")
	cmd.Flags().Bool("no-git", false, "Skip git repository initialization")
	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]
	root, _ := cmd.Flags().GetString("root")
	from, _ := cmd.Flags().GetString("from")
	force, _ := cmd.Flags().GetBool("force")
	noGit, _ := cmd.Flags().GetBool("no-git")

	if filepath.IsAbs(name) || strings.Contains(filepath.Clean(name), "..") {
		return fmt.Errorf("invalid workspace name %q: must be a simple directory name (no absolute paths or ..)", name)
	}

	wsDir := filepath.Join(root, name)
	manifestPath := filepath.Join(wsDir, "workspace.yaml")

	if _, err := os.Stat(wsDir); err == nil && !force {
		return fmt.Errorf("workspace %q already exists (use --force to overwrite)", name)
	}

	// Build manifest data before creating directory to avoid leaving empty dirs on error.
	var data []byte
	var reposRoot string
	switch {
	case from != "":
		src, err := fetchFrom(from)
		if err != nil {
			return fmt.Errorf("reading --from source: %w", err)
		}
		ws, err := manifest.Parse(src)
		if err != nil {
			return fmt.Errorf("invalid manifest from %s: %w", from, err)
		}
		reposRoot = ws.ReposRoot
		data = src
	default:
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("interactive init requires a TTY; use --from to specify a manifest")
		}
		reposRoot = "repos"
		repos, err := interactiveAddRepos(name, reposRoot, nil)
		if err != nil {
			return fmt.Errorf("interactive setup: %w", err)
		}
		var marshalErr error
		data, marshalErr = buildWorkspace(name, reposRoot, repos)
		if marshalErr != nil {
			return fmt.Errorf("building workspace manifest: %w", marshalErr)
		}
	}

	if err := os.MkdirAll(wsDir, 0755); err != nil { //nolint:gosec // workspace dir needs to be world-readable
		return fmt.Errorf("creating workspace directory: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil { //nolint:gosec // manifest file needs to be readable
		return fmt.Errorf("writing manifest: %w", err)
	}

	agentsMDPath := filepath.Join(wsDir, "AGENTS.md")
	agentsMD := generateAgentsMD(name, reposRoot)
	if err := os.WriteFile(agentsMDPath, []byte(agentsMD), 0644); err != nil { //nolint:gosec // AGENTS.md needs to be readable
		return fmt.Errorf("writing AGENTS.md: %w", err)
	}

	if !noGit {
		initGitRepo(cmd, wsDir, reposRoot)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workspace %q created at %s\n", name, wsDir)
	return nil
}

// initGitRepo initializes a git repository in the workspace directory.
// Errors are reported as warnings and do not prevent workspace creation.
func initGitRepo(cmd *cobra.Command, wsDir, reposRoot string) {
	if !git.IsGitInstalled() {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: git is not installed; skipping git initialization\n")
		return
	}

	if git.IsCloned(wsDir) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Git repository already exists in %s; skipping git init\n", wsDir)
		return
	}

	if err := git.Init(wsDir); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: git init failed: %v\n", err)
		return
	}

	gitignoreContent := generateGitignore(reposRoot)
	gitignorePath := filepath.Join(wsDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil { //nolint:gosec // .gitignore needs to be readable
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to write .gitignore: %v\n", err)
		return
	}

	if err := git.Add(wsDir, "workspace.yaml", ".gitignore", "AGENTS.md"); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: git add failed: %v\n", err)
		return
	}

	if err := git.Commit(wsDir, "Initialize workspace"); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: git commit failed: %v\n", err)
		return
	}
}

// generateAgentsMD creates AGENTS.md content with workspace name and repos_root embedded.
func generateAgentsMD(name, reposRoot string) string {
	return fmt.Sprintf(`# %s — agentws workspace

This workspace is managed by **agentws**, a multi-repo workspace manager for coding agents.

## Directory structure

| Path | Description |
|---|---|
| `+"`workspace.yaml`"+` | Workspace manifest — defines repos, branches, and settings |
| `+"`workspace.lock.yaml`"+` | Lock file — pinned commit SHAs (created by `+"`agentws pin`"+`) |
| `+"`%s/`"+` | Root directory where all repositories are cloned |

## Quick reference

| Command | Description |
|---|---|
| `+"`agentws sync`"+` | Clone or update all repos to match the manifest |
| `+"`agentws status`"+` | Show each repo's branch, HEAD, and dirty state |
| `+"`agentws add`"+` | Add a new repo to the workspace interactively |
| `+"`agentws pin`"+` | Snapshot current HEADs into `+"`workspace.lock.yaml`"+` |
| `+"`agentws branches`"+` | List branches across all repos |
| `+"`agentws checkout --branch <b>`"+` | Switch all repos to the given branch |
| `+"`agentws start <ticket> <slug>`"+` | Create a feature branch across all repos |
| `+"`agentws run -- <cmd>`"+` | Run a command in the workspace root |

## Typical workflow

`+"```"+`sh
agentws sync            # clone / update repos
agentws status          # verify state
# ... make changes ...
agentws pin             # lock current commits
`+"```"+`
`, name, reposRoot)
}

// generateGitignore creates .gitignore content with the repos directory excluded.
func generateGitignore(reposRoot string) string {
	dir := reposRoot
	if dir == "" {
		dir = "repos"
	}
	// Ensure trailing slash for directory pattern.
	dir = strings.TrimSuffix(dir, "/") + "/"
	return dir + "\n"
}

// fetchFrom reads manifest content from a local path or repo#path format.
// For repo#path, it uses `git archive` to fetch a single file from a remote repo.
func fetchFrom(src string) ([]byte, error) {
	repo, path, ok := strings.Cut(src, "#")
	if !ok {
		// Local file.
		return os.ReadFile(src) //nolint:gosec // user-provided --from path
	}

	// Remote: git archive --remote=<repo> HEAD <path> | tar -xO <path>
	// For HTTPS/SSH repos, use shallow clone + read instead (git archive --remote
	// is not universally supported by all hosts).
	tmpDir, err := os.MkdirTemp("", "agentws-from-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cmd := exec.Command("git", "clone", "--depth", "1", "--no-checkout", repo, tmpDir) //nolint:gosec // repo URL from user-provided --from flag
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cloning %s: %w", repo, err)
	}

	// Checkout just the single file.
	checkout := exec.Command("git", "checkout", "HEAD", "--", path) //nolint:gosec // path from user-provided --from flag
	checkout.Dir = tmpDir
	checkout.Stderr = os.Stderr
	if err := checkout.Run(); err != nil {
		return nil, fmt.Errorf("checking out %s from %s: %w", path, repo, err)
	}

	return os.ReadFile(filepath.Join(tmpDir, path)) //nolint:gosec // path from user-provided --from flag
}
