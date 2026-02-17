package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fbkclanna/agentws/internal/manifest"
)

func TestRunInit_fromLocalFile(t *testing.T) {
	dir := t.TempDir()

	// Create a source manifest.
	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: imported
repos_root: repos
repos:
  - id: svc
    url: git@github.com:org/svc.git
    path: repos/svc
`)
	if err := os.WriteFile(src, data, 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "imported", "--from", src})

	if err := root.Execute(); err != nil {
		t.Fatalf("init --from failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(dir, "imported", "workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "imported" {
		t.Errorf("name = %q, want %q", ws.Name, "imported")
	}
	if len(ws.Repos) != 1 || ws.Repos[0].ID != "svc" {
		t.Errorf("unexpected repos: %+v", ws.Repos)
	}

	// Verify git repo was initialized.
	wsDir := filepath.Join(dir, "imported")
	if _, err := os.Stat(filepath.Join(wsDir, ".git")); err != nil {
		t.Errorf("expected .git directory: %v", err)
	}

	// Verify .gitignore contains repos_root.
	gitignoreData, err := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignoreData), "repos/") {
		t.Errorf(".gitignore should contain repos/, got: %s", gitignoreData)
	}

	// Verify AGENTS.md exists and contains the workspace name.
	agentsMD, err := os.ReadFile(filepath.Join(wsDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agentsMD), "imported") {
		t.Errorf("AGENTS.md should contain workspace name 'imported', got: %s", agentsMD)
	}
}

func TestRunInit_alreadyExists(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "myws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "myws"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when workspace already exists")
	}
}

func TestRunInit_force(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "myws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Use --from to avoid interactive mode.
	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: myws
repos_root: repos
repos:
  - id: app
    url: git@github.com:org/app.git
    path: repos/app
`)
	if err := os.WriteFile(src, data, 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "myws", "--force", "--from", src})

	if err := root.Execute(); err != nil {
		t.Fatalf("init --force failed: %v", err)
	}

	manifestPath := filepath.Join(wsDir, "workspace.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest not created with --force: %v", err)
	}

	// Verify git repo was initialized.
	if _, err := os.Stat(filepath.Join(wsDir, ".git")); err != nil {
		t.Errorf("expected .git directory with --force: %v", err)
	}

	// Verify AGENTS.md exists.
	if _, err := os.Stat(filepath.Join(wsDir, "AGENTS.md")); err != nil {
		t.Errorf("expected AGENTS.md with --force: %v", err)
	}
}

func TestRunInit_noGit(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: norepo
repos_root: repos
repos:
  - id: svc
    url: git@github.com:org/svc.git
    path: repos/svc
`)
	if err := os.WriteFile(src, data, 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "norepo", "--from", src, "--no-git"})

	if err := root.Execute(); err != nil {
		t.Fatalf("init --no-git failed: %v", err)
	}

	wsDir := filepath.Join(dir, "norepo")
	if _, err := os.Stat(filepath.Join(wsDir, ".git")); err == nil {
		t.Error("expected .git to NOT exist with --no-git")
	}
	if _, err := os.Stat(filepath.Join(wsDir, ".gitignore")); err == nil {
		t.Error("expected .gitignore to NOT exist with --no-git")
	}
	// workspace.yaml should still exist.
	if _, err := os.Stat(filepath.Join(wsDir, "workspace.yaml")); err != nil {
		t.Errorf("workspace.yaml should exist: %v", err)
	}
	// AGENTS.md should exist even without git.
	if _, err := os.Stat(filepath.Join(wsDir, "AGENTS.md")); err != nil {
		t.Errorf("AGENTS.md should exist even with --no-git: %v", err)
	}
}

func TestRunInit_forceWithExistingGitRepo(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "myws")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Pre-initialize a git repo with a commit.
	gitInit := exec.Command("git", "init", "-b", "main")
	gitInit.Dir = wsDir
	if err := gitInit.Run(); err != nil {
		t.Fatalf("pre-init git: %v", err)
	}
	gitConfig1 := exec.Command("git", "config", "user.email", "test@test.com")
	gitConfig1.Dir = wsDir
	_ = gitConfig1.Run()
	gitConfig2 := exec.Command("git", "config", "user.name", "Test")
	gitConfig2.Dir = wsDir
	_ = gitConfig2.Run()

	dummyFile := filepath.Join(wsDir, "existing.txt")
	if err := os.WriteFile(dummyFile, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = wsDir
	_ = gitAdd.Run()
	gitCommit := exec.Command("git", "commit", "-m", "pre-existing commit")
	gitCommit.Dir = wsDir
	if err := gitCommit.Run(); err != nil {
		t.Fatalf("pre-existing commit: %v", err)
	}

	// Record pre-existing HEAD.
	headBefore := exec.Command("git", "rev-parse", "HEAD")
	headBefore.Dir = wsDir
	beforeOut, err := headBefore.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	headSHA := strings.TrimSpace(string(beforeOut))

	// Run init --force.
	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: myws
repos_root: repos
repos:
  - id: app
    url: git@github.com:org/app.git
    path: repos/app
`)
	if err := os.WriteFile(src, data, 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "myws", "--force", "--from", src})
	if err := root.Execute(); err != nil {
		t.Fatalf("init --force with existing git: %v", err)
	}

	// HEAD should be unchanged (git init was skipped).
	headAfter := exec.Command("git", "rev-parse", "HEAD")
	headAfter.Dir = wsDir
	afterOut, err := headAfter.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD after: %v", err)
	}
	if strings.TrimSpace(string(afterOut)) != headSHA {
		t.Errorf("HEAD changed: before=%s after=%s", headSHA, strings.TrimSpace(string(afterOut)))
	}
}

func TestRunInit_customReposRoot(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: custom
repos_root: src/vendor
repos:
  - id: lib
    url: git@github.com:org/lib.git
    path: src/vendor/lib
`)
	if err := os.WriteFile(src, data, 0644); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "custom", "--from", src})

	if err := root.Execute(); err != nil {
		t.Fatalf("init with custom repos_root failed: %v", err)
	}

	wsDir := filepath.Join(dir, "custom")
	gitignoreData, err := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignoreData), "src/vendor/") {
		t.Errorf(".gitignore should contain custom repos_root 'src/vendor/', got: %s", gitignoreData)
	}
}

func TestGenerateAgentsMD(t *testing.T) {
	tests := []struct {
		name      string
		wsName    string
		reposRoot string
		wantName  string
		wantRoot  string
	}{
		{"default", "my-workspace", "repos", "my-workspace", "repos/"},
		{"custom root", "proj", "vendor/src", "proj", "vendor/src/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateAgentsMD(tt.wsName, tt.reposRoot)
			if !strings.Contains(got, tt.wantName) {
				t.Errorf("generateAgentsMD() should contain workspace name %q", tt.wantName)
			}
			if !strings.Contains(got, tt.wantRoot) {
				t.Errorf("generateAgentsMD() should contain repos_root %q", tt.wantRoot)
			}
		})
	}
}

func TestRunInit_fromLocalFile_withBaseRef(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: with-base
repos_root: repos
defaults:
  base_ref: develop
repos:
  - id: svc
    url: git@github.com:org/svc.git
    path: repos/svc
    ref: develop
    base_ref: develop
`)
	if err := os.WriteFile(src, data, 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "with-base", "--from", src})

	if err := root.Execute(); err != nil {
		t.Fatalf("init --from failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(dir, "with-base", "workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if ws.Defaults.BaseRef != "develop" {
		t.Errorf("defaults.base_ref = %q, want %q", ws.Defaults.BaseRef, "develop")
	}
	if len(ws.Repos) != 1 {
		t.Fatalf("repos count = %d, want 1", len(ws.Repos))
	}
	if ws.Repos[0].BaseRef != "develop" {
		t.Errorf("repos[0].base_ref = %q, want %q", ws.Repos[0].BaseRef, "develop")
	}
}

func TestRunInit_baseRefFlag(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "source.yaml")
	data := []byte(`version: 1
name: flagtest
repos_root: repos
repos:
  - id: svc
    url: git@github.com:org/svc.git
    path: repos/svc
`)
	if err := os.WriteFile(src, data, 0644); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	// --base-ref flag should be ignored with --from (source manifest takes priority).
	root := newRootCmd()
	root.SetArgs([]string{"--root", dir, "init", "flagtest", "--from", src, "--base-ref", "ignored"})

	if err := root.Execute(); err != nil {
		t.Fatalf("init --from --base-ref failed: %v", err)
	}

	ws, err := manifest.Load(filepath.Join(dir, "flagtest", "workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// --from source has no base_ref, so it should remain empty (--base-ref is ignored with --from).
	if ws.Defaults.BaseRef != "" {
		t.Errorf("defaults.base_ref = %q, want empty (--base-ref ignored with --from)", ws.Defaults.BaseRef)
	}
}

func TestGenerateGitignore(t *testing.T) {
	tests := []struct {
		name      string
		reposRoot string
		want      string
	}{
		{"default", "repos", "repos/\n"},
		{"empty falls back to repos", "", "repos/\n"},
		{"custom path", "vendor/src", "vendor/src/\n"},
		{"already has trailing slash", "libs/", "libs/\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateGitignore(tt.reposRoot)
			if got != tt.want {
				t.Errorf("generateGitignore(%q) = %q, want %q", tt.reposRoot, got, tt.want)
			}
		})
	}
}
