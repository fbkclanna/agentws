package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]
	root, _ := cmd.Flags().GetString("root")
	from, _ := cmd.Flags().GetString("from")
	force, _ := cmd.Flags().GetBool("force")

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
	switch {
	case from != "":
		src, err := fetchFrom(from)
		if err != nil {
			return fmt.Errorf("reading --from source: %w", err)
		}
		if _, err := manifest.Parse(src); err != nil {
			return fmt.Errorf("invalid manifest from %s: %w", from, err)
		}
		data = src
	default:
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("interactive init requires a TTY; use --from to specify a manifest")
		}
		reposRoot := "repos"
		repos, err := interactiveAddRepos(name, reposRoot)
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

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workspace %q created at %s\n", name, wsDir)
	return nil
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
