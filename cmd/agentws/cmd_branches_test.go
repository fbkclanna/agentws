package main

import (
	"bytes"
	"encoding/json"
	"testing"
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
