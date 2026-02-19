package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fbkclanna/agentws/internal/git"
)

func TestRunBranches_table(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "branches"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("branches failed: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestRunBranches_json(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "branches", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("branches --json failed: %v", err)
	}

	var infos []branchInfo
	if err := json.Unmarshal(buf.Bytes(), &infos); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(infos) != 1 {
		t.Errorf("expected 1 entry, got %d", len(infos))
	}
}

func TestRunBranches_onlyFilter(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "branches", "--json", "--only", "backend"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("branches --json --only backend failed: %v", err)
	}

	var infos []branchInfo
	if err := json.Unmarshal(buf.Bytes(), &infos); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(infos) != 1 {
		t.Errorf("expected 1 entry, got %d", len(infos))
	}
}

func TestRunBranches_notCloned(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 2)

	// sync only backend; frontend remains uncloned
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync", "--only", "backend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --only backend failed: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "branches", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("branches --json failed: %v", err)
	}

	var infos []branchInfo
	if err := json.Unmarshal(buf.Bytes(), &infos); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	found := false
	for _, bi := range infos {
		if bi.Repo == "frontend" {
			found = true
			if bi.Branch != "(not cloned)" {
				t.Errorf("expected branch '(not cloned)' for frontend, got %q", bi.Branch)
			}
		}
	}
	if !found {
		t.Error("expected frontend entry in JSON output")
	}
}

func TestRunBranches_detachedHead(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	repoDir := filepath.Join(wsDir, "repos", "backend")
	sha, err := git.HeadCommit(repoDir)
	if err != nil {
		t.Fatalf("HeadCommit failed: %v", err)
	}
	if err := git.Checkout(repoDir, sha); err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "branches", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("branches --json failed: %v", err)
	}

	var infos []branchInfo
	if err := json.Unmarshal(buf.Bytes(), &infos); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(infos))
	}
	if infos[0].Branch != "(detached)" {
		t.Errorf("expected branch '(detached)', got %q", infos[0].Branch)
	}
}

func TestRunBranches_dirtyRepo(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	repoDir := filepath.Join(wsDir, "repos", "backend")
	if err := os.WriteFile(filepath.Join(repoDir, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("failed to create dirty file: %v", err)
	}

	var buf bytes.Buffer
	root2 := newRootCmd()
	root2.SetOut(&buf)
	root2.SetArgs([]string{"--root", wsDir, "branches", "--json"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("branches --json failed: %v", err)
	}

	var infos []branchInfo
	if err := json.Unmarshal(buf.Bytes(), &infos); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(infos))
	}
	if !infos[0].Dirty {
		t.Error("expected dirty to be true")
	}
}
