package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fbkclanna/agentws/internal/git"
)

// initLocalRepo initializes a local git repository with an initial commit.
// It creates the directory, runs git init, creates a README.md, and commits.
func initLocalRepo(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := git.Init(dir); err != nil {
		return fmt.Errorf("git init in %s: %w", dir, err)
	}

	readmePath := filepath.Join(dir, "README.md")
	repoName := filepath.Base(dir)
	content := fmt.Sprintf("# %s\n", repoName)
	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("creating README.md: %w", err)
	}

	if err := git.Add(dir, "README.md"); err != nil {
		return fmt.Errorf("staging README.md: %w", err)
	}

	if err := git.Commit(dir, "Initial commit"); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	return nil
}
