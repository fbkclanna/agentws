package main

import (
	"testing"
)

func TestRunRun_basicEcho(t *testing.T) {
	// run command has DisableFlagParsing, so --root cannot be passed via args.
	// Set root flag directly instead.
	dir := t.TempDir()
	root := newRootCmd()
	if err := root.PersistentFlags().Set("root", dir); err != nil {
		t.Fatal(err)
	}
	root.SetArgs([]string{"run", "--", "echo", "hello"})
	if err := root.Execute(); err != nil {
		t.Fatalf("run -- echo hello failed: %v", err)
	}
}

func TestRunRun_noArgs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when no args given to run")
	}
}

func TestRunRun_onlyDashDash(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"run", "--"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when only -- given to run")
	}
}
