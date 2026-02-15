package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fbkclanna/agentws/internal/lock"
	"github.com/fbkclanna/agentws/internal/manifest"
)

// Context holds the resolved paths and loaded config for a workspace.
type Context struct {
	Root         string
	ManifestPath string
	LockPath     string
	Manifest     *manifest.Workspace
	Lock         *lock.File // may be nil
}

// Load resolves workspace paths and loads the manifest (and lock if present).
func Load(root string) (*Context, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace root: %w", err)
	}

	manifestPath := filepath.Join(root, "workspace.yaml")
	lockPath := filepath.Join(root, "workspace.lock.yaml")

	ws, err := manifest.Load(manifestPath)
	if err != nil {
		return nil, err
	}

	ctx := &Context{
		Root:         root,
		ManifestPath: manifestPath,
		LockPath:     lockPath,
		Manifest:     ws,
	}

	if _, statErr := os.Stat(lockPath); statErr == nil {
		lf, err := lock.Load(lockPath)
		if err != nil {
			return nil, err
		}
		ctx.Lock = lf
	}

	return ctx, nil
}

// RepoDir returns the absolute path for a repo within the workspace.
func (c *Context) RepoDir(repo manifest.Repo) string {
	return filepath.Join(c.Root, repo.Path)
}

// Strategy represents how to handle dirty working trees.
type Strategy string

const (
	StrategySafe  Strategy = "safe"
	StrategyStash Strategy = "stash"
	StrategyReset Strategy = "reset"
)

// ParseStrategy parses a strategy string, defaulting to "safe".
func ParseStrategy(s string) (Strategy, error) {
	switch Strategy(s) {
	case StrategySafe, "":
		return StrategySafe, nil
	case StrategyStash:
		return StrategyStash, nil
	case StrategyReset:
		return StrategyReset, nil
	default:
		return "", fmt.Errorf("unknown strategy: %q (must be safe, stash, or reset)", s)
	}
}
