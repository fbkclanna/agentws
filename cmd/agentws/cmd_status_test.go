package main

import (
	"bytes"
	"encoding/json"
	"testing"
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
