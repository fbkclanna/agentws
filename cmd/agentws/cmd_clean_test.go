package main

import (
	"os"
	"testing"
)

func TestRunClean_requiresForce(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"clean"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error without --force")
	}
}

func TestRunClean_removesDirectory(t *testing.T) {
	wsDir, _ := setupWorkspace(t, 1)

	// Sync to create repo dirs.
	root := newRootCmd()
	root.SetArgs([]string{"--root", wsDir, "sync"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Clean.
	root2 := newRootCmd()
	root2.SetArgs([]string{"--root", wsDir, "clean", "--force"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("clean --force failed: %v", err)
	}

	if _, err := os.Stat(wsDir); !os.IsNotExist(err) {
		t.Error("workspace directory should have been removed")
	}
}
