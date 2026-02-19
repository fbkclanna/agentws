package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
)

func TestRunStatus_table(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	// Sync first.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Run status.
	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "status"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("status failed: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestRunStatus_json(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "status", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var statuses []repoStatus
	if err := json.Unmarshal(buf.Bytes(), &statuses); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(statuses) != 1 {
		t.Errorf("expected 1 status entry, got %d", len(statuses))
	}
	if !statuses[0].Cloned {
		t.Error("repo should be cloned")
	}
}

func TestRunStatus_dirtyRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Make dirty.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "status", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var statuses []repoStatus
	if err := json.Unmarshal(buf.Bytes(), &statuses); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(statuses))
	}
	if !statuses[0].Dirty {
		t.Error("expected dirty: true for repo with uncommitted file")
	}
}

func TestRunStatus_unclonedRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	// Sync only backend.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--only", "backend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "status", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var statuses []repoStatus
	if err := json.Unmarshal(buf.Bytes(), &statuses); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.ID == "frontend" {
			found = true
			if s.Cloned {
				t.Error("frontend should not be cloned")
			}
		}
	}
	if !found {
		t.Error("frontend not found in status output")
	}
}

func TestRunStatus_lockDiff(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// Sync and pin.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "pin"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("pin failed: %v", err)
	}

	// Create a local commit to advance HEAD past the pinned commit.
	dir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(dir, "local.txt"), []byte("local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := git.Add(dir, "local.txt"); err != nil {
		t.Fatal(err)
	}
	if err := git.Commit(dir, "local commit"); err != nil {
		t.Fatal(err)
	}

	// Status should show lock_diff since HEAD moved past pinned commit.
	var buf bytes.Buffer
	root3 := newRootCmd()
	root3.SetOut(&buf)
	root3.SetArgs([]string{"--root", wsDir, "status", "--json"})
	if err := root3.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var statuses []repoStatus
	if err := json.Unmarshal(buf.Bytes(), &statuses); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(statuses))
	}
	if statuses[0].LockDiff == "" {
		t.Error("expected non-empty lock_diff when HEAD differs from pinned commit")
	}
}
