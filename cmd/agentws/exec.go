package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fbkclanna/agentws/internal/manifest"
)

// execCmd runs a PostSync command safely (no shell expansion).
func execCmd(repoDir string, ps manifest.PostSync) error {
	if len(ps.Cmd) == 0 {
		return fmt.Errorf("empty cmd")
	}

	dir := repoDir
	if ps.WorkDir != "" {
		dir = filepath.Join(repoDir, ps.WorkDir)
	}

	cmd := exec.Command(ps.Cmd[0], ps.Cmd[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
