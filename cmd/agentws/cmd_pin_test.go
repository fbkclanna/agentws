package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/lock"
)

func TestRunPin_createsLock(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	// Sync first.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Pin.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "pin"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("pin failed: %v", err)
	}

	lockPath := filepath.Join(wsDir, "workspace.lock.yaml")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file not created: %v", err)
	}

	lf, err := lock.Load(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(lf.Repos) != 2 {
		t.Errorf("expected 2 pinned repos, got %d", len(lf.Repos))
	}
	for _, r := range lf.Repos {
		if r.Commit == "" {
			t.Error("commit should not be empty")
		}
	}
}

func TestRunPin_skipsUncloned(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	// Sync only backend so frontend remains uncloned.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--only", "backend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --only backend failed: %v", err)
	}

	// Pin.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "pin"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("pin failed: %v", err)
	}

	lockPath := filepath.Join(wsDir, "workspace.lock.yaml")
	lf, err := lock.Load(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(lf.Repos) != 1 {
		t.Errorf("expected 1 pinned repo (only cloned backend), got %d", len(lf.Repos))
	}
	if _, ok := lf.Repos["backend"]; !ok {
		t.Error("expected backend to be in lock file")
	}
}
